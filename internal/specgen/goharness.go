package specgen

import (
	"bytes"
	"fmt"
	"go/format"

	"goalchemy/internal/contracts"
)

func init() { harnessGenerators["go"] = goHarness }

func goType(t *contracts.TypeExpr) string {
	switch t.Kind {
	case "slice":
		return "[]" + goType(t.Elem)
	case "array":
		return fmt.Sprintf("[%d]%s", t.Len, goType(t.Elem))
	case "map":
		return "rt.Map[" + goType(t.Key) + ", " + goType(t.Elem) + "]"
	case "pointer":
		return "*" + goType(t.Elem)
	}
	return t.Name
}

func goDecode(t *contracts.TypeExpr, raw string) string {
	switch t.Kind {
	case "slice":
		return fmt.Sprintf("DecSlice(%s, func(r json.RawMessage) %s { return %s })", raw, goType(t.Elem), goDecode(t.Elem, "r"))
	case "map":
		return fmt.Sprintf("DecMap(%s, func(r json.RawMessage) %s { return %s }, func(r json.RawMessage) %s { return %s })",
			raw, goType(t.Key), goDecode(t.Key, "r"), goType(t.Elem), goDecode(t.Elem, "r"))
	}
	switch t.Name {
	case "error":
		return "DecError(" + raw + ")"
	case "bool":
		return "DecBool(" + raw + ")"
	case "string":
		return "DecString(" + raw + ")"
	}
	return fmt.Sprintf("DecInt[%s](%s)", t.Name, raw)
}

func goEncode(t *contracts.TypeExpr, v string) string {
	switch t.Kind {
	case "array":
		return fmt.Sprintf("EncArray(%s[:], func(e %s) any { return %s })", v, goType(t.Elem), goEncode(t.Elem, "e"))
	case "slice":
		return fmt.Sprintf("EncSlice(%s, func(e %s) any { return %s })", v, goType(t.Elem), goEncode(t.Elem, "e"))
	case "map":
		return fmt.Sprintf("EncMap(%s, func(e %s) any { return %s }, func(e %s) any { return %s })",
			v, goType(t.Key), goEncode(t.Key, "e"), goType(t.Elem), goEncode(t.Elem, "e"))
	}
	switch t.Name {
	case "error":
		return "EncError(" + v + ")"
	case "bool":
		return "EncBool(" + v + ")"
	case "string":
		return "EncString(" + v + ")"
	}
	return "EncInt(" + v + ")"
}

func goHarness(cat *contracts.Catalog, t *contracts.Target) ([]byte, error) {
	cases, err := collectCases(cat, t)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "// %s\n\npackage main\n\nimport (\n\t\"encoding/json\"\n\n\trt \"goalchemy/targets/go/runtime\"\n)\n\n", Marker)
	b.WriteString("var _ json.RawMessage\n\nvar cases = map[string]func(h *H) []any{\n")
	for _, ci := range cases {
		fmt.Fprintf(&b, "\t%q: func(h *H) []any {\n", ci.ID())
		for _, l := range ci.lets {
			name := "v_" + l.name
			switch l.kind {
			case "view":
				fmt.Fprintf(&b, "\t\t%s := View(v_%s, h.Let(%q))\n", name, l.of, l.name)
			case "ref":
				fmt.Fprintf(&b, "\t\t%s := v_%s\n", name, l.of)
			default:
				fmt.Fprintf(&b, "\t\t%s := %s\n", name, goDecode(l.t, fmt.Sprintf("h.Let(%q)", l.name)))
			}
			fmt.Fprintf(&b, "\t\t_ = %s\n", name)
		}
		var args []string
		for _, a := range ci.c.Call {
			args = append(args, "v_"+a)
		}
		types := map[string]string{}
		for k, v := range ci.args {
			types[k] = goType(v)
		}
		call := expand(ci.impl.Harness, args, types)
		var rs []string
		for i := range ci.outs {
			rs = append(rs, fmt.Sprintf("r%d", i))
		}
		if len(rs) > 0 {
			fmt.Fprintf(&b, "\t\t%s := %s\n", join(rs), call)
		} else {
			fmt.Fprintf(&b, "\t\t%s\n", call)
		}
		for _, a := range ci.c.Expect.After {
			var lt *contracts.TypeExpr
			for _, l := range ci.lets {
				if l.name == a.Name {
					lt = l.t
				}
			}
			fmt.Fprintf(&b, "\t\th.After(%q, %s)\n", a.Name, goEncode(lt, "v_"+a.Name))
		}
		var encs []string
		for i, o := range ci.outs {
			encs = append(encs, goEncode(o, rs[i]))
		}
		fmt.Fprintf(&b, "\t\treturn []any{%s}\n\t},\n", join(encs))
	}
	b.WriteString("}\n")
	return format.Source(b.Bytes())
}

func join(xs []string) string {
	s := ""
	for i, x := range xs {
		if i > 0 {
			s += ", "
		}
		s += x
	}
	return s
}
