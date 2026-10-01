package specgen

import (
	"bytes"
	"fmt"
	"strings"

	"goalchemy/internal/contracts"
)

func init() { harnessGenerators["rust"] = rustHarness }

func rustZeroFn(t *contracts.TypeExpr) string {
	switch t.Kind {
	case "slice":
		return "zero_slice"
	case "map", "pointer", "chan":
		return "zero_nil"
	}
	switch t.Name {
	case "bool":
		return "zero_bool"
	case "string":
		return "zero_string"
	case "error", "any", "func()", "context.Context", "struct{}":
		return "zero_nil"
	}
	return "zero_int"
}

func rustDecode(t *contracts.TypeExpr, raw string) string {
	switch t.Kind {
	case "chan":
		return fmt.Sprintf("dec_chan(%s, &|r: &J| %s, %s)", raw, rustDecode(t.Elem, "r"), rustZeroFn(t.Elem))
	case "pointer":
		switch t.Elem.Name {
		case "sync.WaitGroup":
			return "new_waitgroup()"
		case "sync.Mutex":
			return "new_mutex()"
		}
		panic("rust harness: pointer to " + t.Elem.Name)
	case "slice":
		return fmt.Sprintf("dec_slice(%s, &|r: &J| %s, %s)", raw, rustDecode(t.Elem, "r"), rustZeroFn(t.Elem))
	case "map":
		return fmt.Sprintf("dec_map(%s, &|r: &J| %s, &|r: &J| %s)", raw, rustDecode(t.Key, "r"), rustDecode(t.Elem, "r"))
	}
	switch t.Name {
	case "func()", "struct{}":
		return "V::Nil"
	case "context.Context":
		return "std_context_background()"
	case "error":
		return "dec_error(" + raw + ")"
	case "bool":
		return "dec_bool(" + raw + ")"
	case "string":
		return "dec_string(" + raw + ")"
	}
	return fmt.Sprintf("dec_int(%s, %q)", raw, tsKind(t))
}

func rustEncode(t *contracts.TypeExpr, v string) string {
	switch t.Kind {
	case "chan":
		return fmt.Sprintf("enc_chan(%s, &|e: &V| %s)", v, rustEncode(t.Elem, "e"))
	case "slice":
		return fmt.Sprintf("enc_slice(%s, &|e: &V| %s)", v, rustEncode(t.Elem, "e"))
	case "array":
		return fmt.Sprintf("enc_array(%s, &|e: &V| %s)", v, rustEncode(t.Elem, "e"))
	case "map":
		return fmt.Sprintf("enc_map(%s, &|e: &V| %s, &|e: &V| %s)", v, rustEncode(t.Key, "e"), rustEncode(t.Elem, "e"))
	}
	switch t.Name {
	case "context.Context", "context.CancelFunc", "struct{}":
		return "enc_zero(" + v + ")"
	case "error":
		return "enc_error(" + v + ")"
	case "bool":
		return "enc_bool(" + v + ")"
	case "string":
		return "enc_string(" + v + ")"
	}
	return fmt.Sprintf("enc_int(%s, %q)", v, tsKind(t))
}

func rustHarness(cat *contracts.Catalog, t *contracts.Target) ([]byte, error) {
	cases, err := collectCases(cat, t)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "// %s\nuse crate::codec::*;\nuse crate::json::J;\nuse crate::rt::*;\nuse crate::H;\n\n", Marker)
	var names []string
	for n, ci := range cases {
		fn := fmt.Sprintf("case_%d", n)
		names = append(names, fmt.Sprintf("    (%q, %s),\n", ci.ID(), fn))
		fmt.Fprintf(&b, "fn %s(h: &H) -> Vec<J> {\n", fn)
		var lets []string
		for _, l := range ci.lets {
			name := "v_" + l.name
			lets = append(lets, name+".clone()")
			switch l.kind {
			case "view":
				fmt.Fprintf(&b, "    let %s = view(&v_%s, h.let_(%q));\n", name, l.of, l.name)
			case "ref":
				fmt.Fprintf(&b, "    let %s = v_%s.clone();\n", name, l.of)
			default:
				fmt.Fprintf(&b, "    let %s = %s;\n", name, rustDecode(l.t, fmt.Sprintf("h.let_(%q)", l.name)))
			}
		}
		fmt.Fprintf(&b, "    let _roots = temp_root(&[%s]);\n", strings.Join(lets, ", "))
		var args []string
		for k := range ci.c.Call {
			args = append(args, fmt.Sprintf("a%d.clone()", k))
		}
		for k, a := range ci.c.Call {
			fmt.Fprintf(&b, "    let a%d = v_%s.clone();\n", k, a)
		}
		vars := map[string]string{}
		for k, v := range ci.args {
			vars[k+".kind"] = tsKind(v)
			vars[k+".zerofn"] = rustZeroFn(v)
			vars[k+".unsigned"] = fmt.Sprint(tsKind(v) == "u64")
			vars[k+".count"] = "count"
			vars[k+".u"] = ""
			if tsKind(v) == "u64" {
				vars[k+".count"] = "countu"
				vars[k+".u"] = "u"
			}
		}
		call := unsetU.ReplaceAllString(expand(ci.impl.Harness, args, vars), "")
		switch {
		case strings.HasPrefix(call, "await:"):
			fmt.Fprintf(&b, "    let rv = run_isolated(move |t: &Rc<Task>| { %s; });\n", strings.TrimSpace(strings.TrimPrefix(call, "await:")))
			for i := range ci.outs {
				fmt.Fprintf(&b, "    let r%d = rv[%d].clone();\n", i, i)
			}
		case len(ci.outs) == 0:
			fmt.Fprintf(&b, "    %s;\n", call)
		case len(ci.outs) == 1:
			fmt.Fprintf(&b, "    let r0 = %s;\n", call)
		default:
			fmt.Fprintf(&b, "    let rv = %s;\n", call)
			for i := range ci.outs {
				fmt.Fprintf(&b, "    let r%d = rv.at(%d);\n", i, i)
			}
		}
		for _, a := range ci.c.Expect.After {
			var lt *contracts.TypeExpr
			for _, l := range ci.lets {
				if l.name == a.Name {
					lt = l.t
				}
			}
			fmt.Fprintf(&b, "    h.after(%q, %s);\n", a.Name, rustEncode(lt, "&v_"+a.Name))
		}
		var encs []string
		for i, o := range ci.outs {
			encs = append(encs, rustEncode(o, fmt.Sprintf("&r%d", i)))
		}
		fmt.Fprintf(&b, "    vec![%s]\n}\n\n", join(encs))
	}
	b.WriteString("pub static CASES: &[(&str, fn(&H) -> Vec<J>)] = &[\n")
	for _, n := range names {
		b.WriteString(n)
	}
	b.WriteString("];\n")
	return b.Bytes(), nil
}
