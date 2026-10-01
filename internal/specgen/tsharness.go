package specgen

import (
	"bytes"
	"fmt"

	"goalchemy/internal/contracts"
)

func init() { harnessGenerators["typescript"] = tsHarness }

func tsKind(t *contracts.TypeExpr) string {
	bits, signed, ok := contracts.IntInfo(t.Name)
	if !ok || t.Kind != "name" {
		return ""
	}
	if signed {
		return fmt.Sprintf("i%d", bits)
	}
	return fmt.Sprintf("u%d", bits)
}

func tsZero(t *contracts.TypeExpr) string {
	switch t.Kind {
	case "slice":
		return "rt.NIL"
	case "map", "pointer":
		return "null"
	}
	switch t.Name {
	case "bool":
		return "false"
	case "string":
		return `""`
	case "error", "any":
		return "null"
	}
	if bits, _, _ := contracts.IntInfo(t.Name); bits == 64 {
		return "0n"
	}
	return "0"
}

func tsDecode(t *contracts.TypeExpr, raw string) string {
	switch t.Kind {
	case "slice":
		return fmt.Sprintf("decSlice(%s, (r: any) => %s)", raw, tsDecode(t.Elem, "r"))
	case "map":
		return fmt.Sprintf("decMap(%s, (r: any) => %s, (r: any) => %s)", raw, tsDecode(t.Key, "r"), tsDecode(t.Elem, "r"))
	}
	switch t.Name {
	case "bool":
		return "decBool(" + raw + ")"
	case "string":
		return "decString(" + raw + ")"
	}
	cast := "number"
	if bits, _, _ := contracts.IntInfo(t.Name); bits == 64 {
		cast = "bigint"
	}
	return fmt.Sprintf("(decInt(%s, %q) as %s)", raw, tsKind(t), cast)
}

func tsEncode(t *contracts.TypeExpr, v string) string {
	switch t.Kind {
	case "slice":
		return fmt.Sprintf("encSlice(%s, (e: any) => %s)", v, tsEncode(t.Elem, "e"))
	case "array":
		return fmt.Sprintf("encArray(%s, (e: any) => %s)", v, tsEncode(t.Elem, "e"))
	case "map":
		return fmt.Sprintf("encMap(%s, (e: any) => %s, (e: any) => %s)", v, tsEncode(t.Key, "e"), tsEncode(t.Elem, "e"))
	}
	switch t.Name {
	case "bool":
		return "encBool(" + v + ")"
	case "string":
		return "encString(" + v + ")"
	}
	return "encInt(" + v + ")"
}

func tsHarness(cat *contracts.Catalog, t *contracts.Target) ([]byte, error) {
	cases, err := collectCases(cat, t)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "// %s\nimport * as rt from \"./runtime_index.ts\";\n", Marker)
	b.WriteString("import { decInt, decBool, decString, decSlice, decMap, view, encInt, encBool, encString, encSlice, encArray, encMap, type H } from \"./codec.ts\";\n\n")
	b.WriteString("export const cases: Record<string, (h: H) => unknown[]> = {\n")
	for _, ci := range cases {
		fmt.Fprintf(&b, "  %q: (h: H) => {\n", ci.ID())
		for _, l := range ci.lets {
			name := "v_" + l.name
			switch l.kind {
			case "view":
				fmt.Fprintf(&b, "    const %s = view(v_%s, h.let(%q));\n", name, l.of, l.name)
			case "ref":
				fmt.Fprintf(&b, "    const %s = v_%s;\n", name, l.of)
			default:
				fmt.Fprintf(&b, "    const %s = %s;\n", name, tsDecode(l.t, fmt.Sprintf("h.let(%q)", l.name)))
			}
		}
		var args []string
		for _, a := range ci.c.Call {
			args = append(args, "v_"+a)
		}
		vars := map[string]string{}
		for k, v := range ci.args {
			vars[k+".kind"] = tsKind(v)
			vars[k+".zero"] = tsZero(v)
		}
		call := expand(ci.impl.Harness, args, vars)
		switch len(ci.outs) {
		case 0:
			fmt.Fprintf(&b, "    %s;\n", call)
		case 1:
			fmt.Fprintf(&b, "    const r0 = %s;\n", call)
		default:
			var rs []string
			for i := range ci.outs {
				rs = append(rs, fmt.Sprintf("r%d", i))
			}
			fmt.Fprintf(&b, "    const [%s] = %s;\n", join(rs), call)
		}
		for _, a := range ci.c.Expect.After {
			var lt *contracts.TypeExpr
			for _, l := range ci.lets {
				if l.name == a.Name {
					lt = l.t
				}
			}
			fmt.Fprintf(&b, "    h.after(%q, %s);\n", a.Name, tsEncode(lt, "v_"+a.Name))
		}
		var encs []string
		for i, o := range ci.outs {
			encs = append(encs, tsEncode(o, fmt.Sprintf("r%d", i)))
		}
		fmt.Fprintf(&b, "    return [%s];\n  },\n", join(encs))
	}
	b.WriteString("};\n")
	return b.Bytes(), nil
}
