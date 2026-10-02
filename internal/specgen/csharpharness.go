package specgen

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/eugenioenko/goalchemy/internal/contracts"
)

func init() { harnessGenerators["csharp"] = csHarness }

func csZero(t *contracts.TypeExpr) string {
	switch t.Kind {
	case "slice":
		if tsKind(t.Elem) == "u8" {
			return "Slice.BYTE_NIL"
		}
		return "Slice.NIL"
	case "map", "pointer", "chan":
		return "null"
	}
	switch t.Name {
	case "bool":
		return "false"
	case "string":
		return `""`
	case "error", "any", "func()", "context.Context", "struct{}":
		return "null"
	}
	return "0L"
}

func csDecode(t *contracts.TypeExpr, raw string) string {
	switch t.Kind {
	case "chan":
		return fmt.Sprintf("decChan(%s, r => %s, () => %s)", raw, csDecode(t.Elem, "r"), csZero(t.Elem))
	case "pointer":
		switch t.Elem.Name {
		case "sync.WaitGroup":
			return "new WaitGroup()"
		case "sync.Mutex":
			return "new GoMutex()"
		case "crypto.Key":
			return "(Native.Key)null"
		}
		panic("csharp harness: pointer to " + t.Elem.Name)
	case "slice":
		if tsKind(t.Elem) == "u8" {
			return fmt.Sprintf("decByteSlice(%s)", raw)
		}
		return fmt.Sprintf("decSlice(%s, r => %s, () => %s)", raw, csDecode(t.Elem, "r"), csZero(t.Elem))
	case "map":
		return fmt.Sprintf("decMap(%s, r => %s, r => %s)", raw, csDecode(t.Key, "r"), csDecode(t.Elem, "r"))
	}
	switch t.Name {
	case "func()":
		return "(Fn) null"
	case "struct{}":
		return "(object) null"
	case "context.Context":
		return "R.stdContextBackground()"
	case "error":
		return "decError(" + raw + ")"
	case "bool":
		return "decBool(" + raw + ")"
	case "string":
		return "decString(" + raw + ")"
	}
	return fmt.Sprintf("decInt(%s, %q)", raw, tsKind(t))
}

func csEncode(t *contracts.TypeExpr, v string) string {
	switch t.Kind {
	case "pointer":
		if t.Elem.Name == "crypto.Key" {
			return "encKey(" + v + ")"
		}
	case "chan":
		return fmt.Sprintf("encChan(%s, e => %s)", v, csEncode(t.Elem, "e"))
	case "slice":
		return fmt.Sprintf("encSlice(%s, e => %s)", v, csEncode(t.Elem, "e"))
	case "array":
		return fmt.Sprintf("encArray(%s, e => %s)", v, csEncode(t.Elem, "e"))
	case "map":
		return fmt.Sprintf("encMap(%s, e => %s, e => %s)", v, csEncode(t.Key, "e"), csEncode(t.Elem, "e"))
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
	return fmt.Sprintf("encInt(%s, %q)", v, tsKind(t))
}

func csHarness(cat *contracts.Catalog, t *contracts.Target) ([]byte, error) {
	cases, err := collectCases(cat, t)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "// %s\nusing System.Text.Json.Nodes;\nusing Rt;\n\nstatic partial class Harness\n{\n", Marker)
	var names []string
	for n, ci := range cases {
		fn := fmt.Sprintf("case_%d", n)
		names = append(names, fmt.Sprintf("        [%q] = %s,\n", ci.ID(), fn))
		fmt.Fprintf(&b, "    static JsonArray %s(H h)\n    {\n", fn)
		for _, l := range ci.lets {
			name := "v_" + l.name
			switch l.kind {
			case "view":
				fmt.Fprintf(&b, "        var %s = view(v_%s, h.Let(%q));\n", name, l.of, l.name)
			case "ref":
				fmt.Fprintf(&b, "        var %s = v_%s;\n", name, l.of)
			default:
				fmt.Fprintf(&b, "        var %s = %s;\n", name, csDecode(l.t, fmt.Sprintf("h.Let(%q)", l.name)))
			}
		}
		var args []string
		for _, a := range ci.c.Call {
			args = append(args, "v_"+a)
		}
		vars := map[string]string{}
		for k, v := range ci.args {
			vars[k+".kind"] = tsKind(v)
			vars[k+".zero"] = csZero(v)
			vars[k+".bytes"] = fmt.Sprint(tsKind(v) == "u8")
			vars[k+".unsigned"] = fmt.Sprint(tsKind(v) == "u64")
			vars[k+".count"] = "Ints.count"
			vars[k+".u"] = ""
			if tsKind(v) == "u64" {
				vars[k+".count"] = "Ints.countu"
				vars[k+".u"] = "u"
			}
		}
		call := unsetU.ReplaceAllString(expand(ci.impl.Harness, args, vars), "")
		switch {
		case strings.HasPrefix(call, "host:"):
			fmt.Fprintf(&b, "        object[] rv = host(t => { %s; });\n", strings.TrimSpace(strings.TrimPrefix(call, "host:")))
			for i := range ci.outs {
				fmt.Fprintf(&b, "        var r%d = rv[%d];\n", i, i)
			}
		case strings.HasPrefix(call, "await:"):
			fmt.Fprintf(&b, "        object[] rv = R.runIsolated(t => { %s; });\n", strings.TrimSpace(strings.TrimPrefix(call, "await:")))
			for i := range ci.outs {
				fmt.Fprintf(&b, "        var r%d = rv[%d];\n", i, i)
			}
		case len(ci.outs) == 0:
			fmt.Fprintf(&b, "        %s;\n", call)
		case len(ci.outs) == 1:
			fmt.Fprintf(&b, "        var r0 = %s;\n", call)
		default:
			fmt.Fprintf(&b, "        object[] rv = (object[]) (%s);\n", call)
			for i := range ci.outs {
				fmt.Fprintf(&b, "        var r%d = rv[%d];\n", i, i)
			}
		}
		for _, a := range ci.c.Expect.After {
			var lt *contracts.TypeExpr
			for _, l := range ci.lets {
				if l.name == a.Name {
					lt = l.t
				}
			}
			fmt.Fprintf(&b, "        h.After(%q, %s);\n", a.Name, csEncode(lt, "v_"+a.Name))
		}
		var encs []string
		for i, o := range ci.outs {
			encs = append(encs, csEncode(o, fmt.Sprintf("r%d", i)))
		}
		fmt.Fprintf(&b, "        return new JsonArray(%s);\n    }\n\n", join(encs))
	}
	b.WriteString("    static readonly Dictionary<string, Func<H, JsonArray>> CASES = new()\n    {\n")
	for _, n := range names {
		b.WriteString(n)
	}
	b.WriteString("    };\n}\n")
	return b.Bytes(), nil
}
