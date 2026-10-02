package specgen

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/eugenioenko/goalchemy/internal/contracts"
)

func init() { harnessGenerators["c"] = cHarness }

func cZeroFn(t *contracts.TypeExpr) string {
	if t.Kind == "slice" && (t.Elem.Name == "byte" || t.Elem.Name == "uint8") {
		return "gx_zero_byte_slice"
	}
	return "gx_" + rustZeroFn(t)
}

// cCodec generates named decoder and encoder functions for composite types,
// since C has no closures.
type cCodec struct {
	b     *bytes.Buffer
	names map[string]string
}

func (c *cCodec) fn(key, def string) string {
	if n, ok := c.names[key]; ok {
		return n
	}
	n := fmt.Sprintf("codec_%d", len(c.names))
	c.names[key] = n
	fmt.Fprintf(c.b, def, n)
	return n
}

// dec returns a DecFn for t.
func (c *cCodec) dec(t *contracts.TypeExpr) string {
	switch t.Kind {
	case "chan":
		el := c.dec(t.Elem)
		return c.fn("dec:"+t.String(), "static gx_V %s(J *r) { return dec_chan(r, "+el+", "+cZeroFn(t.Elem)+"); }\n")
	case "slice":
		el := c.dec(t.Elem)
		return c.fn("dec:"+t.String(), "static gx_V %s(J *r) { return dec_slice(r, "+el+", "+cZeroFn(t.Elem)+", "+fmt.Sprint(t.Elem.Name == "byte" || t.Elem.Name == "uint8")+"); }\n")
	case "map":
		k, v := c.dec(t.Key), c.dec(t.Elem)
		return c.fn("dec:"+t.String(), "static gx_V %s(J *r) { return dec_map(r, "+k+", "+v+"); }\n")
	case "pointer":
		switch t.Elem.Name {
		case "sync.WaitGroup":
			return c.fn("dec:wg", "static gx_V %s(J *r) { return gx_new_waitgroup(); }\n")
		case "sync.Mutex":
			return c.fn("dec:mu", "static gx_V %s(J *r) { return gx_new_mutex(); }\n")
		}
		panic("c harness: pointer to " + t.Elem.Name)
	}
	switch t.Name {
	case "func()", "struct{}":
		return "dec_nil"
	case "context.Context":
		return c.fn("dec:ctx", "static gx_V %s(J *r) { return gx_std_context_background(); }\n")
	case "error":
		return "dec_error"
	case "bool":
		return "dec_bool"
	case "string":
		return "dec_string"
	}
	return "dec_int"
}

// enc returns an EncFn for t.
func (c *cCodec) enc(t *contracts.TypeExpr) string {
	switch t.Kind {
	case "chan":
		el := c.enc(t.Elem)
		return c.fn("enc:"+t.String(), "static J *%s(gx_V v) { return enc_chan(v, "+el+"); }\n")
	case "slice":
		el := c.enc(t.Elem)
		return c.fn("enc:"+t.String(), "static J *%s(gx_V v) { return enc_slice(v, "+el+"); }\n")
	case "array":
		el := c.enc(t.Elem)
		return c.fn("enc:"+t.String(), fmt.Sprintf("static J *%%s(gx_V v) { return enc_array(v, %d, %s); }\n", t.Len, el))
	case "map":
		k, v := c.enc(t.Key), c.enc(t.Elem)
		return c.fn("enc:"+t.String(), "static J *%s(gx_V v) { return enc_map(v, "+k+", "+v+"); }\n")
	}
	switch t.Name {
	case "context.Context", "context.CancelFunc", "struct{}":
		return "enc_zero"
	case "error":
		return "enc_error"
	case "bool":
		return "enc_bool"
	case "string":
		return "enc_string"
	}
	if tsKind(t) == "u64" {
		return "enc_int_u"
	}
	return "enc_int_s"
}

func cHarness(cat *contracts.Catalog, t *contracts.Target) ([]byte, error) {
	cases, err := collectCases(cat, t)
	if err != nil {
		return nil, err
	}
	var helpers, body bytes.Buffer
	cc := &cCodec{b: &helpers, names: map[string]string{}}
	var names []string
	for n, ci := range cases {
		fn := fmt.Sprintf("case_%d", n)
		names = append(names, fmt.Sprintf("    {%q, %s},\n", ci.ID(), fn))
		vars := map[string]string{}
		for k, v := range ci.args {
			vars[k+".kind"] = tsKind(v)
			vars[k+".zerofn"] = cZeroFn(v)
			vars[k+".makefn"] = "gx_make_slice"
			vars[k+".makezero"] = ", " + cZeroFn(v)
			if v.Name == "byte" || v.Name == "uint8" {
				vars[k+".makefn"] = "gx_make_byte_slice"
				vars[k+".makezero"] = ""
			}
			vars[k+".unsigned"] = fmt.Sprint(tsKind(v) == "u64")
			vars[k+".count"] = "gx_count"
			vars[k+".u"] = ""
			if tsKind(v) == "u64" {
				vars[k+".count"] = "gx_countu"
				vars[k+".u"] = "u"
			}
		}
		var args []string
		for k := range ci.c.Call {
			args = append(args, fmt.Sprintf("a[%d]", k))
		}
		call := unsetU.ReplaceAllString(expand(ci.impl.Harness, args, vars), "")
		await := strings.HasPrefix(call, "await:")
		if await {
			fmt.Fprintf(&body, "static void prim_%d(gx_Task *t, void *arg) {\n    gx_V *a = arg;\n    %s;\n}\n\n", n, strings.TrimSpace(strings.TrimPrefix(call, "await:")))
		}
		fmt.Fprintf(&body, "static J *%s(H *h) {\n", fn)
		for _, l := range ci.lets {
			name := "v_" + l.name
			switch l.kind {
			case "view":
				fmt.Fprintf(&body, "    gx_V %s = view(v_%s, h_let(h, %q));\n", name, l.of, l.name)
			case "ref":
				fmt.Fprintf(&body, "    gx_V %s = v_%s;\n", name, l.of)
			default:
				fmt.Fprintf(&body, "    gx_V %s = %s(h_let(h, %q));\n", name, cc.dec(l.t), l.name)
			}
		}
		var av []string
		for _, a := range ci.c.Call {
			av = append(av, "v_"+a)
		}
		if len(av) == 0 {
			av = []string{"gx_nil()"}
		}
		fmt.Fprintf(&body, "    gx_V a[] = {%s};\n", strings.Join(av, ", "))
		switch {
		case await:
			fmt.Fprintf(&body, "    gx_V rv[8];\n    gx_run_isolated(prim_%d, a, rv);\n", n)
			for i := range ci.outs {
				fmt.Fprintf(&body, "    gx_V r%d = rv[%d];\n", i, i)
			}
		case len(ci.outs) == 0:
			fmt.Fprintf(&body, "    %s;\n", call)
		case len(ci.outs) == 1:
			fmt.Fprintf(&body, "    gx_V r0 = %s;\n", call)
		default:
			fmt.Fprintf(&body, "    gx_V rt = %s;\n", call)
			for i := range ci.outs {
				fmt.Fprintf(&body, "    gx_V r%d = gx_at(rt, %d);\n", i, i)
			}
		}
		for _, a := range ci.c.Expect.After {
			var lt *contracts.TypeExpr
			for _, l := range ci.lets {
				if l.name == a.Name {
					lt = l.t
				}
			}
			fmt.Fprintf(&body, "    h_after(h, %q, %s(v_%s));\n", a.Name, cc.enc(lt), a.Name)
		}
		body.WriteString("    J *out = j_new(J_ARR);\n")
		for i, o := range ci.outs {
			fmt.Fprintf(&body, "    j_push(out, %s(r%d));\n", cc.enc(o), i)
		}
		body.WriteString("    return out;\n}\n\n")
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "/* %s */\n#include \"codec.h\"\n\n", Marker)
	b.Write(helpers.Bytes())
	b.WriteString("\n")
	b.Write(body.Bytes())
	b.WriteString("const Case CASES[] = {\n")
	for _, n := range names {
		b.WriteString(n)
	}
	fmt.Fprintf(&b, "};\n\nconst size_t NCASES = %d;\n", len(names))
	return b.Bytes(), nil
}
