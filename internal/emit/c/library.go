package c

import (
	"bytes"
	"fmt"
	"github.com/eugenioenko/goalchemy/internal/ir"
	"go/token"
	"sort"
	"strings"
)

func cValueType(t *ir.Type, seen map[*ir.Type]bool) bool {
	if seen[t] {
		return true
	}
	seen[t] = true
	u := t.U()
	switch u.Kind {
	case ir.KBool, ir.KInt, ir.KString, ir.KFloat:
		return true
	case ir.KArray, ir.KSlice:
		return cValueType(u.Elem, seen)
	case ir.KStruct:
		for _, f := range u.Fields {
			if !token.IsExported(f.Name) || f.Embedded || !cValueType(f.Type, seen) {
				return false
			}
		}
		return true
	}
	return false
}
func nativeBytes(t *ir.Type) bool { u := t.U(); return u.Kind == ir.KInt && u.Int == ir.U8 }
func cContext(t *ir.Type) bool    { return t.Kind == ir.KOpaque && t.Name == "context.Context" }
func cError(t *ir.Type) bool      { return t.Kind == ir.KNamed && t.Name == "error" }
func (e *emitter) valueLibrary(out *bytes.Buffer) ([]byte, error) {
	e.use("core.task.spawn")
	e.use("std.context.with_cancel")
	e.use("std.context.with_timeout")
	out.WriteString(`#include "library.h"
extern const gxc_options *gxc_active_options;
#include <stdlib.h>
#include <limits.h>
void goalchemy_wake(void) {gx_library_wake();}
static pthread_mutex_t gxc_serial = PTHREAD_MUTEX_INITIALIZER;
static const gxc_value *gxc_args;
static const gxc_options *gxc_opts;
static gxc_value *gxc_result;
static gxc_error *gxc_failure;
static int gxc_export;
static gx_V gxc_context;
static void gxc_invalid(void) { gxc_failure->kind=5; gx_host_fault("invalid native library value"); }
typedef struct gxc_allocation {void *pointer;struct gxc_allocation *next;} gxc_allocation;
static gxc_allocation *gxc_allocations;
static size_t gxc_output_bytes,gxc_output_nodes;static unsigned gxc_output_depth;
static void gxc_allocation_release(bool failed) {while(gxc_allocations){gxc_allocation *node=gxc_allocations;gxc_allocations=node->next;if(failed)free(node->pointer);free(node);}}
static void *gxc_alloc(size_t n) { if(n>256u*1024u*1024u-gxc_output_bytes)gx_host_fault("native result byte budget");gxc_output_bytes+=n; void *p=calloc(n?n:1,1); if(!p) gx_host_fault("native library allocation");gxc_allocation *node=malloc(sizeof(*node));if(!node){free(p);gx_host_fault("native library allocation tracking");}node->pointer=p;node->next=gxc_allocations;gxc_allocations=node;return p; }
static gxc_value gxc_bytes_out(const void *p,size_t n) {
 gxc_value v={.kind=GXC_BYTES,.length=n}; v.bytes=gxc_alloc(n); if(n) memcpy(v.bytes,p,n); return v;
}
static gxc_value gxc_list_out(gxc_kind kind,size_t n) {
 if(n>SIZE_MAX/sizeof(gxc_value)) gx_host_fault("native result length");
 gxc_value v={.kind=kind,.length=n}; v.items=gxc_alloc(n*sizeof(*v.items));
 if(kind==GXC_RECORD) v.names=gxc_alloc(n*sizeof(*v.names)); return v;
}
static void gxc_name(gxc_value *v,size_t i,const char *name) {
 size_t n=strlen(name); v->names[i]=gxc_alloc(n+1); memcpy(v->names[i],name,n+1);
}
static void gxc_poll(void) {
 if(gxc_opts && gxc_opts->canceled && atomic_load(gxc_opts->canceled))
  gx_cancel_ctx(gxc_context,gx_context_canceled());
}
`)
	seen := map[*ir.Type]bool{}
	var convert func(*ir.Type)
	convert = func(t *ir.Type) {
		u := t.U()
		if seen[u] {
			return
		}
		seen[u] = true
		e.localProto("static gx_V gxc_in_%s(const gxc_value *v)", e.names.Type(u, ""))
		e.localProto("static gxc_value gxc_out_%s(gx_V v)", e.names.Type(u, ""))
		e.localProto("static gxc_value gxc_out_body_%s(gx_V v)", e.names.Type(u, ""))
		switch u.Kind {
		case ir.KStruct:
			for _, f := range u.Fields {
				convert(f.Type)
			}
		case ir.KArray, ir.KSlice:
			convert(u.Elem)
		}
		fmt.Fprintf(out, "static gx_V gxc_in_%s(const gxc_value *v) {\n", e.names.Type(u, ""))
		fmt.Fprintf(out, " if(v->kind==GXC_NIL) return %s;\n", e.zero(t))
		switch u.Kind {
		case ir.KFloat:
			fmt.Fprintf(out, " if(v->kind!=GXC_FLOAT) gxc_invalid(); return gx_float(gx_round_float(v->floating,%d));\n", u.FloatBits)
		case ir.KBool:
			out.WriteString(" if(v->kind!=GXC_BOOL || (v->integer!=0 && v->integer!=1)) gxc_invalid(); return gx_bool(v->integer);\n")
		case ir.KInt:
			out.WriteString(" if(v->kind!=GXC_INT) gxc_invalid();\n")
			if u.Int != ir.I64 && u.Int != ir.U64 {
				fmt.Fprintf(out, " if((int64_t)(%s)v->integer!=v->integer) gxc_invalid();\n", fmt.Sprintf("%sint%d_t", map[bool]string{true: "", false: "u"}[u.Int.Signed()], u.Int.Bits()))
			}
			out.WriteString(" return gx_int(v->integer);\n")
		case ir.KString:
			out.WriteString(" if(v->kind!=GXC_BYTES) gxc_invalid(); return gx_str((const char *)v->bytes,v->length);\n")
		case ir.KStruct:
			out.WriteString(" if(v->kind!=GXC_RECORD) gxc_invalid();\n")
			fmt.Fprintf(out, " gx_V r=gx_new_vals(%d,NULL);\n", len(u.Fields))
			for i, f := range u.Fields {
				fmt.Fprintf(out, " gx_fset(r,%d,gxc_in_%s(gxc_field(v,%q)));\n", i, e.names.Type(f.Type.U(), ""), f.Name)
			}
			out.WriteString(" return r;\n")
		case ir.KArray, ir.KSlice:
			if u.Kind == ir.KArray {
				fmt.Fprintf(out, " if(v->length!=%d) gxc_invalid();\n", u.Len)
			}
			if nativeBytes(u.Elem) {
				out.WriteString(" if(v->kind!=GXC_BYTES) gxc_invalid(); uint8_t *b=gx_alloc_bytes(v->length); if(v->length) memcpy(b,v->bytes,v->length);\n")
				if u.Kind == ir.KArray {
					out.WriteString(" gx_V r=gx_obj(b); r.pad=1; return r;\n")
				} else {
					out.WriteString(" return gx_byte_slice(b,(uint32_t)v->length,(uint32_t)v->length);\n")
				}
			} else {
				out.WriteString(" if(v->kind!=GXC_LIST) gxc_invalid(); gx_V *a=gx_alloc_vals(v->length);\n")
				fmt.Fprintf(out, " for(size_t i=0;i<v->length;i++) a[i]=gxc_in_%s(&v->items[i]);\n", e.names.Type(u.Elem.U(), ""))
				if u.Kind == ir.KArray {
					out.WriteString(" return gx_obj(a);\n")
				} else {
					out.WriteString(" return gx_slice(a,(uint32_t)v->length,(uint32_t)v->length);\n")
				}
			}
		}
		out.WriteString("}\n")
		fmt.Fprintf(out, "static gxc_value gxc_out_%s(gx_V v) {if(gxc_output_depth>=64 || ++gxc_output_nodes>1000000)gx_host_fault(\"native result tree budget\");gxc_output_depth++;gxc_value r=gxc_out_body_%s(v);gxc_output_depth--;return r;}\nstatic gxc_value gxc_out_body_%s(gx_V v) {\n", e.names.Type(u, ""), e.names.Type(u, ""), e.names.Type(u, ""))
		switch u.Kind {
		case ir.KFloat:
			out.WriteString(" return (gxc_value){.kind=GXC_FLOAT,.floating=gx_f(v)};\n")
		case ir.KBool:
			out.WriteString(" return (gxc_value){.kind=GXC_BOOL,.integer=gx_b(v)};\n")
		case ir.KInt:
			out.WriteString(" return (gxc_value){.kind=GXC_INT,.integer=gx_i(v)};\n")
		case ir.KString:
			out.WriteString(" return gxc_bytes_out(gx_sbytes(v),gx_slen(v));\n")
		case ir.KStruct:
			fmt.Fprintf(out, " gxc_value r=gxc_list_out(GXC_RECORD,%d);\n", len(u.Fields))
			for i, f := range u.Fields {
				fmt.Fprintf(out, " gxc_name(&r,%d,%q); r.items[%d]=gxc_out_%s(gx_fld(v,%d));\n", i, f.Name, i, e.names.Type(f.Type.U(), ""), i)
			}
			out.WriteString(" return r;\n")
		case ir.KArray, ir.KSlice:
			n := "v.l"
			if u.Kind == ir.KArray {
				n = fmt.Sprint(u.Len)
			}
			if nativeBytes(u.Elem) {
				fmt.Fprintf(out, " return gxc_bytes_out(gx_bytes(v),%s);\n", n)
			} else {
				fmt.Fprintf(out, " gxc_value r=gxc_list_out(GXC_LIST,%s); for(size_t i=0;i<r.length;i++) r.items[i]=gxc_out_%s(gx_vals(v)[i]); return r;\n", n, e.names.Type(u.Elem.U(), ""))
			}
		}
		out.WriteString("}\n")
	}
	for _, f := range e.p.Exports {
		for _, t := range f.Sig.Params {
			if cContext(t) {
				continue
			}
			if !cValueType(t, map[*ir.Type]bool{}) {
				return nil, fmt.Errorf("unsupported C library input %s", ir.TypeString(t))
			}
			convert(t)
		}
		for _, t := range f.Sig.Results {
			if cError(t) {
				continue
			}
			if !cValueType(t, map[*ir.Type]bool{}) {
				return nil, fmt.Errorf("unsupported C library output %s", ir.TypeString(t))
			}
			convert(t)
		}
	}
	var errors []*ir.Type
	for _, t := range e.p.Types.All {
		if t.Kind == ir.KPointer && t.Elem.U().Kind == ir.KStruct && cValueType(t.Elem, map[*ir.Type]bool{}) && e.tds[t] {
			errors = append(errors, t)
			convert(t.Elem)
		}
	}
	sort.Slice(errors, func(i, j int) bool { return errors[i].ID < errors[j].ID })
	out.WriteString("static void gxc_source_error(gx_V v) {\n gxc_failure->kind=1; gx_Buf b={0}; gx_format_panic_value(v,&b); gxc_failure->message=gxc_bytes_out(b.b,b.n);\n")
	for _, t := range errors {
		fmt.Fprintf(out, " if(gx_is_type(v,&td_%s)) { gxc_failure->fields=gxc_out_%s(gx_unboxed(v)); return; }\n", e.names.Type(t, ""), e.names.Type(t.Elem.U(), ""))
	}
	out.WriteString("}\n")
	fmt.Fprintf(out, "static void gxc_init_source(void) { init_zero_globals(); f_%s(); }\n", e.names.Symbol(e.p.Init.Sym))
	out.WriteString("static gx_V gxc_call_source(void) { gx_V c=gx_std_context_with_cancel(gx_background()); gxc_context=gx_at(c,0);\n if(gxc_opts && gxc_opts->timeout_nanoseconds>0) gxc_context=gx_at(gx_std_context_with_timeout(gxc_context,gx_int(gxc_opts->timeout_nanoseconds)),0);\n gxc_poll(); switch(gxc_export) {\n")
	for k, f := range e.p.Exports {
		var as []string
		index := 0
		for _, t := range f.Sig.Params {
			if cContext(t) {
				as = append(as, "gxc_context")
			} else {
				as = append(as, fmt.Sprintf("gxc_in_%s(&gxc_args[%d])", e.names.Type(t.U(), ""), index))
				index++
			}
		}
		if f.MaySuspend {
			fmt.Fprintf(out, " case %d: return f_%s(%s);\n", k, e.names.Symbol(f.Sym), strings.Join(as, ","))
		} else {
			vec, n := vec(as)
			fmt.Fprintf(out, " case %d: return gx_sync_frame(gx_func(-1,w_%s,0,NULL),%d,%s,%d);\n", k, e.names.Symbol(f.Sym), n, vec, len(f.Sig.Results))
		}
	}
	out.WriteString(" } gx_host_fault(\"unknown export\"); }\nstatic void gxc_root_step(gx_Task *t,gx_Frame *f) {if(!f->pc){f->pc=1;gx_call(t,gxc_call_source());return;}gx_ret(t,f); }\nstatic gx_V gxc_entry(void) {return gx_vframe(gx_new_frame(0,gxc_root_step,NULL));}\nstatic void gxc_finish(gx_Task *t) { switch(gxc_export) {\n")
	for k, f := range e.p.Exports {
		n := len(f.Sig.Results)
		hasError := n > 0 && cError(f.Sig.Results[n-1])
		if hasError {
			n--
		}
		fmt.Fprintf(out, " case %d:\n", k)
		if hasError {
			fmt.Fprintf(out, " if(!gx_is_nil(gx_rv(t,%d))) {gxc_source_error(gx_rv(t,%d)); break;}\n", n, n)
		}
		if n == 1 {
			fmt.Fprintf(out, " *gxc_result=gxc_out_%s(gx_rv(t,0));\n", e.names.Type(f.Sig.Results[0].U(), ""))
		} else if n > 1 {
			fmt.Fprintf(out, " *gxc_result=gxc_list_out(GXC_LIST,%d);\n", n)
			for i := 0; i < n; i++ {
				fmt.Fprintf(out, " gxc_result->items[%d]=gxc_out_%s(gx_rv(t,%d));\n", i, e.names.Type(f.Sig.Results[i].U(), ""), i)
			}
		}
		out.WriteString(" break;\n")
	}
	out.WriteString(` } }
int goalchemy_invoke(const char *name,const gxc_value *args,size_t count,const gxc_options *opts,gxc_value *result,gxc_error *error) {
 if(!result || !error) return 5;
 memset(result,0,sizeof(*result)); memset(error,0,sizeof(*error));
 if(gxc_callback_thread) return error->kind=3;
 if(!name || count>256 || (count && !args)) return error->kind=5;
 int which=-1; size_t expected=0;
`)
	for k, f := range e.p.Exports {
		n := 0
		for _, t := range f.Sig.Params {
			if !cContext(t) {
				n++
			}
		}
		name := f.Name[strings.LastIndex(f.Name, ".")+1:]
		fmt.Fprintf(out, " if(!strcmp(name,%q)) {which=%d;expected=%d;}\n", name, k, n)
	}
	out.WriteString(` if(which<0 || count!=expected) return error->kind=5;
 gxc_value *snapshot=calloc(count?count:1,sizeof(*snapshot)); if(!snapshot) return error->kind=3;
 int rc=0; for(size_t i=0;i<count;i++) {rc=gxc_value_copy(&snapshot[i],&args[i]);if(rc)break;}
 if(rc) {for(size_t i=0;i<count;i++) gxc_value_free(&snapshot[i]);free(snapshot);return error->kind=rc==3?3:5;}
 while(pthread_mutex_trylock(&gxc_serial)) {
  if(opts && opts->canceled && atomic_load(opts->canceled)) {rc=6;break;}
  struct timespec pause={0,1000000};nanosleep(&pause,NULL);
 }
 if(!rc) {
  if(opts && opts->canceled && atomic_load(opts->canceled)) {rc=6;} else {
   gxc_args=snapshot;gxc_opts=opts;gxc_active_options=opts;gxc_result=result;gxc_failure=error;gxc_export=which;
   gxc_output_bytes=0;gxc_output_nodes=0;gxc_output_depth=0;
   uint8_t *report=NULL;size_t n=0;
   rc=gx_run_library_host(gxc_init_source,gxc_entry,gxc_finish,gxc_poll,&report,&n);
   gxc_allocation_release(rc!=0);
   if(rc) {int kind=error->kind?error->kind:rc;memset(result,0,sizeof(*result));memset(error,0,sizeof(*error));error->kind=kind;error->message=(gxc_value){.kind=GXC_BYTES,.bytes=report,.length=n};}
   else free(report);
   gxc_context=gx_nil();gxc_args=NULL;gxc_opts=NULL;gxc_active_options=NULL;gxc_result=NULL;gxc_failure=NULL;
  }
  pthread_mutex_unlock(&gxc_serial);
 }
 for(size_t i=0;i<count;i++)gxc_value_free(&snapshot[i]);free(snapshot);
 if(rc && !error->kind)error->kind=rc;
 return error->kind;
}
`)
	return []byte("/* Generated owned C value library API. */\n#include \"library.h\"\n"), nil
}
