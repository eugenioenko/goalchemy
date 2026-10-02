/* Go panics as longjmp to the innermost handler, deferred calls, and
 * recover. */
#include "gx.h"

#include <stdio.h>

gx_Handler *gx_handler;
gx_Panic *gx_thrown;

gx_Panic *gx_new_panic(gx_V v) {
    gx_Panic *p = GC_MALLOC(sizeof(gx_Panic));
    p->value = v;
    return p;
}

_Noreturn void gx_raise(gx_Panic *p) {
    gx_Handler *h = gx_handler;
    if (!h) gx_report_panic(p);
    gx_handler = h->prev;
    gx_source_depth = h->source_depth;
    gx_thrown = p;
    longjmp(h->jb, 1);
}

_Noreturn void gx_throw(gx_V v) {
    if (v.t == GX_NIL) v = gx_boxv(&GX_PANIC_NIL_ERROR, gx_nil());
    gx_raise(gx_new_panic(v));
}

void gx_chain_panic(gx_Panic *p, gx_Panic *earlier) {
    if (earlier && p != earlier && !p->prev) p->prev = earlier;
}

static gx_V runtime_error_text(gx_V *e, gx_V *a, int n) {
    (void)e;
    (void)n;
    gx_Buf b = {0};
    gx_buf_put(&b, "runtime error: ", 15);
    gx_buf_put(&b, a[0].u.p, a[0].l);
    return gx_str((const char *)b.b, b.n);
}

static gx_V plain_error_text(gx_V *e, gx_V *a, int n) {
    (void)e;
    (void)n;
    return a[0];
}

static gx_V no_value(gx_V *e, gx_V *a, int n) {
    (void)e;
    (void)a;
    (void)n;
    return gx_nil();
}

static gx_V panic_nil_text(gx_V *e, gx_V *a, int n) {
    (void)e;
    (void)a;
    (void)n;
    return gx_cstr("runtime error: panic called with nil argument");
}

static void key_basic(gx_V v, gx_Buf *out) { gx_vkey(v, out); }

static const gx_Method runtime_error_methods[] = {{"Error", -1, runtime_error_text}, {"RuntimeError", -1, no_value}};
static const gx_Method plain_error_methods[] = {{"Error", -1, plain_error_text}, {"RuntimeError", -1, no_value}};
static const gx_Method panic_nil_methods[] = {{"Error", -1, panic_nil_text}, {"RuntimeError", -1, no_value}};

const gx_TypeDesc GX_RUNTIME_ERROR = {0x7fff0001, "runtime.Error", "runtime_error", gx_eq_basic, key_basic, runtime_error_methods, 2, "", true};
const gx_TypeDesc GX_PLAIN_ERROR = {0x7fff0002, "runtime.plainError", "runtime_error", gx_eq_basic, key_basic, plain_error_methods, 2, "", true};
const gx_TypeDesc GX_TYPE_ASSERTION_ERROR = {0x7fff0003, "*runtime.TypeAssertionError", "runtime_error", gx_eq_basic, key_basic, plain_error_methods, 2, "", true};
const gx_TypeDesc GX_PANIC_NIL_ERROR = {0x7fff0004, "*runtime.PanicNilError", "runtime_error", gx_eq_basic, key_basic, panic_nil_methods, 2, "", true};
const gx_TypeDesc GX_STRING_TYPE = {0x7fff0005, "string", "string", gx_eq_basic, key_basic, NULL, 0, "string", true};

gx_V gx_runtime_error(const char *msg) { return gx_boxv(&GX_RUNTIME_ERROR, gx_cstr(msg)); }

_Noreturn void gx_runtime_panic(const char *msg) { gx_throw(gx_runtime_error(msg)); }

_Noreturn void gx_plain_panic(const char *msg) { gx_throw(gx_boxv(&GX_PLAIN_ERROR, gx_cstr(msg))); }

_Noreturn void gx_panic_nil_deref(void) { gx_runtime_panic("invalid memory address or nil pointer dereference"); }

size_t gx_idx(gx_V i, size_t len) {
    char buf[96];
    int64_t x = i.u.i;
    if (x < 0) {
        snprintf(buf, sizeof buf, "index out of range [%lld]", (long long)x);
        gx_runtime_panic(buf);
    }
    if ((uint64_t)x >= len) {
        snprintf(buf, sizeof buf, "index out of range [%lld] with length %zu", (long long)x, len);
        gx_runtime_panic(buf);
    }
    return (size_t)x;
}

size_t gx_idxu(gx_V i, size_t len) {
    char buf[96];
    uint64_t x = (uint64_t)i.u.i;
    if (x >= len) {
        snprintf(buf, sizeof buf, "index out of range [%llu] with length %zu", (unsigned long long)x, len);
        gx_runtime_panic(buf);
    }
    return (size_t)x;
}

_Noreturn void gx_assert_panic(gx_V x, const char *iface, const char *target, const char *missing) {
    char buf[512];
    const gx_TypeDesc *t = gx_dyn_type(x);
    if (!t)
        snprintf(buf, sizeof buf, "interface conversion: %s is nil, not %s", iface, target);
    else if (missing)
        snprintf(buf, sizeof buf, "interface conversion: %s is not %s: missing method %s", t->name, target, missing);
    else
        snprintf(buf, sizeof buf, "interface conversion: %s is %s, not %s", iface, t->name, target);
    gx_throw(gx_boxv(&GX_TYPE_ASSERTION_ERROR, gx_cstr(buf)));
}

gx_Deferred *gx_defer(gx_Deferred *list, gx_V f, int n, const gx_V *args, int64_t fid, bool start) {
    gx_Deferred *d = GC_MALLOC(sizeof(gx_Deferred));
    d->f = f;
    d->n = n;
    d->args = gx_alloc_vals((size_t)n);
    if (n) memcpy(d->args, args, (size_t)n * sizeof(gx_V));
    d->fid = fid;
    d->start = start;
    d->next = list;
    return d;
}

/* Runs a sequential function's deferred calls after its body finished,
 * normally or by panic p, and re-raises a panic still active afterwards. */
void gx_run_defers(gx_Deferred *list, gx_Panic *p) {
    gx_Panic *volatile panicking = p;
    gx_Deferred *volatile d = list;
    while (d) {
        gx_Deferred *cur = d;
        d = d->next;
        gx_Task *t = gx_cur_task();
        gx_Panic *saved_panic = t->cur_panic;
        int64_t saved_target = t->defer_target;
        t->cur_panic = panicking;
        t->defer_target = cur->fid;
        gx_Handler h;
        if (GX_TRY(h)) {
            t->cur_panic = saved_panic;
            t->defer_target = saved_target;
            gx_chain_panic(gx_thrown, panicking);
            panicking = gx_thrown;
            continue;
        }
        gx_callv(cur->f, cur->n, cur->args);
        GX_END(h);
        t->cur_panic = saved_panic;
        t->defer_target = saved_target;
        if (panicking && panicking->recovered) panicking = NULL;
    }
    if (panicking) gx_raise(panicking);
}

gx_V gx_recover(int64_t fid) {
    gx_Task *t = gx_cur_task();
    gx_Panic *p = t->cur_panic;
    if (!p || p->recovered || t->defer_target != fid) return gx_nil();
    p->recovered = true;
    return p->value;
}
