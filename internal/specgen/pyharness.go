package specgen

import (
	"bytes"
	"fmt"
	"strings"

	"goalchemy/internal/contracts"
)

func init() { harnessGenerators["python"] = pyHarness }

func pyZero(t *contracts.TypeExpr) string {
	switch t.Kind {
	case "slice":
		return "rt.NIL"
	case "map", "pointer", "chan":
		return "None"
	}
	switch t.Name {
	case "bool":
		return "False"
	case "string":
		return `b""`
	case "error", "any", "func()", "context.Context":
		return "None"
	}
	return "0"
}

func pyDecode(t *contracts.TypeExpr, raw string) string {
	switch t.Kind {
	case "chan":
		return fmt.Sprintf("dec_chan(%s, lambda r: %s, lambda: %s)", raw, pyDecode(t.Elem, "r"), pyZero(t.Elem))
	case "pointer":
		return "rt." + t.Elem.Name[strings.Index(t.Elem.Name, ".")+1:] + "()"
	case "slice":
		return fmt.Sprintf("dec_slice(%s, lambda r: %s, lambda: %s)", raw, pyDecode(t.Elem, "r"), pyZero(t.Elem))
	case "map":
		return fmt.Sprintf("dec_map(%s, lambda r: %s, lambda r: %s)", raw, pyDecode(t.Key, "r"), pyDecode(t.Elem, "r"))
	}
	switch t.Name {
	case "func()":
		return "None"
	case "context.Context":
		return "rt.std_context_background()"
	case "struct{}":
		return "None"
	case "error":
		return "dec_error(" + raw + ")"
	case "bool":
		return "dec_bool(" + raw + ")"
	case "string":
		return "dec_string(" + raw + ")"
	}
	return fmt.Sprintf("dec_int(%s, %q)", raw, tsKind(t))
}

func pyEncode(t *contracts.TypeExpr, v string) string {
	switch t.Kind {
	case "chan":
		return fmt.Sprintf("enc_chan(%s, lambda e: %s)", v, pyEncode(t.Elem, "e"))
	case "slice":
		return fmt.Sprintf("enc_slice(%s, lambda e: %s)", v, pyEncode(t.Elem, "e"))
	case "array":
		return fmt.Sprintf("enc_array(%s, lambda e: %s)", v, pyEncode(t.Elem, "e"))
	case "map":
		return fmt.Sprintf("enc_map(%s, lambda e: %s, lambda e: %s)", v, pyEncode(t.Key, "e"), pyEncode(t.Elem, "e"))
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
	return "enc_int(" + v + ")"
}

func pyHarness(cat *contracts.Catalog, t *contracts.Target) ([]byte, error) {
	cases, err := collectCases(cat, t)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "# %s\n# flake8: noqa\nfrom codec import *\nfrom codec import rt\n\n", Marker)
	var names []string
	for n, ci := range cases {
		fn := fmt.Sprintf("case_%d", n)
		names = append(names, fmt.Sprintf("    %q: %s,\n", ci.ID(), fn))
		fmt.Fprintf(&b, "\ndef %s(h):\n", fn)
		for _, l := range ci.lets {
			name := "v_" + l.name
			switch l.kind {
			case "view":
				fmt.Fprintf(&b, "    %s = view(v_%s, h.let(%q))\n", name, l.of, l.name)
			case "ref":
				fmt.Fprintf(&b, "    %s = v_%s\n", name, l.of)
			default:
				fmt.Fprintf(&b, "    %s = %s\n", name, pyDecode(l.t, fmt.Sprintf("h.let(%q)", l.name)))
			}
		}
		var args []string
		for _, a := range ci.c.Call {
			args = append(args, "v_"+a)
		}
		vars := map[string]string{}
		for k, v := range ci.args {
			vars[k+".kind"] = tsKind(v)
			vars[k+".zero"] = pyZero(v)
		}
		call := expand(ci.impl.Harness, args, vars)
		var rs []string
		for i := range ci.outs {
			rs = append(rs, fmt.Sprintf("r%d", i))
		}
		switch {
		case strings.HasPrefix(call, "await:"):
			fmt.Fprintf(&b, "    rv = rt.run_isolated(lambda t: %s)\n", strings.TrimSpace(strings.TrimPrefix(call, "await:")))
			for i := range ci.outs {
				fmt.Fprintf(&b, "    r%d = rv[%d]\n", i, i)
			}
		case len(rs) == 0:
			fmt.Fprintf(&b, "    %s\n", call)
		case len(rs) == 1:
			fmt.Fprintf(&b, "    r0 = %s\n", call)
		default:
			fmt.Fprintf(&b, "    %s = %s\n", join(rs), call)
		}
		for _, a := range ci.c.Expect.After {
			var lt *contracts.TypeExpr
			for _, l := range ci.lets {
				if l.name == a.Name {
					lt = l.t
				}
			}
			fmt.Fprintf(&b, "    h.after(%q, %s)\n", a.Name, pyEncode(lt, "v_"+a.Name))
		}
		var encs []string
		for i, o := range ci.outs {
			encs = append(encs, pyEncode(o, fmt.Sprintf("r%d", i)))
		}
		fmt.Fprintf(&b, "    return [%s]\n\n", join(encs))
	}
	b.WriteString("\nCASES = {\n")
	for _, n := range names {
		b.WriteString(n)
	}
	b.WriteString("}\n")
	return b.Bytes(), nil
}
