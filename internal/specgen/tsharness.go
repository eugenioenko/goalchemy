package specgen

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/eugenioenko/goalchemy/internal/contracts"
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
		if tsKind(t.Elem) == "u8" {
			return "rt.BYTE_NIL"
		}
		return "rt.NIL"
	case "map", "pointer":
		return "null"
	}
	switch t.Name {
	case "bool":
		return "false"
	case "string":
		return `""`
	case "error", "any", "func()":
		return "null"
	}
	if bits, _, _ := contracts.IntInfo(t.Name); bits == 64 {
		return "0n"
	}
	return "0"
}

func tsDecode(t *contracts.TypeExpr, raw string) string {
	switch t.Kind {
	case "chan":
		return fmt.Sprintf("decChan(%s, (r: any) => %s, () => %s)", raw, tsDecode(t.Elem, "r"), tsZero(t.Elem))
	case "pointer":
		if t.Elem.Name == "crypto.Key" {
			return "decKey(" + raw + ")"
		}
		return "new rt." + t.Elem.Name[strings.Index(t.Elem.Name, ".")+1:] + "()"
	case "slice":
		return fmt.Sprintf("decSlice(%s, (r: any) => %s, %t)", raw, tsDecode(t.Elem, "r"), tsKind(t.Elem) == "u8")
	case "map":
		return fmt.Sprintf("decMap(%s, (r: any) => %s, (r: any) => %s)", raw, tsDecode(t.Key, "r"), tsDecode(t.Elem, "r"))
	}
	switch t.Name {
	case "func()":
		return "null"
	case "context.Context":
		return "rt.stdContextBackground()"
	case "struct{}":
		return "{}"
	case "error":
		return "decError(" + raw + ")"
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
	case "pointer":
		if t.Elem.Name == "crypto.Key" {
			return "encKey(" + v + ")"
		}
	case "chan":
		return fmt.Sprintf("encChan(%s, (e: any) => %s)", v, tsEncode(t.Elem, "e"))
	case "slice":
		return fmt.Sprintf("encSlice(%s, (e: any) => %s)", v, tsEncode(t.Elem, "e"))
	case "array":
		return fmt.Sprintf("encArray(%s, (e: any) => %s)", v, tsEncode(t.Elem, "e"))
	case "map":
		return fmt.Sprintf("encMap(%s, (e: any) => %s, (e: any) => %s)", v, tsEncode(t.Key, "e"), tsEncode(t.Elem, "e"))
	}
	switch t.Name {
	case "context.Context", "context.CancelFunc", "struct{}":
		return "encZero(" + v + ")"
	case "error":
		return "encError(" + v + ")"
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
	b.WriteString("import { decInt, decBool, decString, decSlice, decMap, decError, decChan, view, encInt, encBool, encString, encSlice, encArray, encMap, encError, encChan, encZero, newSpawnCheck, harnessSelect2, decKey, encKey, harnessHost, type H } from \"./codec.ts\";\n\n")
	b.WriteString("export const cases: Record<string, (h: H) => unknown[] | Promise<unknown[]>> = {\n")
	for _, ci := range cases {
		async := ""
		if strings.HasPrefix(ci.impl.Harness, "host:") {
			async = "async "
		}
		fmt.Fprintf(&b, "  %q: %s(h: H) => {\n", ci.ID(), async)
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
			vars[k+".bytes"] = fmt.Sprint(tsKind(v) == "u8")
			vars[k+".zero"] = tsZero(v)
		}
		call := expand(ci.impl.Harness, args, vars)
		if strings.HasPrefix(call, "host:") {
			fmt.Fprintf(&b, "    const rv = await harnessHost((t:any)=>{ %s; });\n", strings.TrimSpace(strings.TrimPrefix(call, "host:")))
			for i := range ci.outs {
				fmt.Fprintf(&b, "    const r%d:any=rv[%d];\n", i, i)
			}
		} else if strings.HasPrefix(call, "await:") {
			fmt.Fprintf(&b, "    const rv = rt.runIsolated((t: any) => { %s; });\n", strings.TrimSpace(strings.TrimPrefix(call, "await:")))
			for i := range ci.outs {
				fmt.Fprintf(&b, "    const r%d: any = rv[%d];\n", i, i)
			}
		} else {
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
