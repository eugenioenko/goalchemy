package specgen

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"goalchemy/internal/contracts"
)

func init() { harnessGenerators["java"] = javaHarness }

var unsetU = regexp.MustCompile(`\{[A-Z]\.u\}`)

func javaZero(t *contracts.TypeExpr) string {
	switch t.Kind {
	case "slice":
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

func javaDecode(t *contracts.TypeExpr, raw string) string {
	switch t.Kind {
	case "chan":
		return fmt.Sprintf("decChan(%s, r -> %s, () -> %s)", raw, javaDecode(t.Elem, "r"), javaZero(t.Elem))
	case "pointer":
		switch t.Elem.Name {
		case "sync.WaitGroup":
			return "new StdSyncWaitgroupAdd.WaitGroup()"
		case "sync.Mutex":
			return "new StdSyncMutexLock.Mutex()"
		}
		panic("java harness: pointer to " + t.Elem.Name)
	case "slice":
		return fmt.Sprintf("decSlice(%s, r -> %s, () -> %s)", raw, javaDecode(t.Elem, "r"), javaZero(t.Elem))
	case "map":
		return fmt.Sprintf("decMap(%s, r -> %s, r -> %s)", raw, javaDecode(t.Key, "r"), javaDecode(t.Elem, "r"))
	}
	switch t.Name {
	case "func()":
		return "(Fn) null"
	case "struct{}":
		return "(Object) null"
	case "context.Context":
		return "StdContextBackground.stdContextBackground()"
	case "error":
		return "decError(" + raw + ")"
	case "bool":
		return "decBool(" + raw + ")"
	case "string":
		return "decString(" + raw + ")"
	}
	return fmt.Sprintf("decInt(%s, %q)", raw, tsKind(t))
}

func javaEncode(t *contracts.TypeExpr, v string) string {
	switch t.Kind {
	case "chan":
		return fmt.Sprintf("encChan(%s, e -> %s)", v, javaEncode(t.Elem, "e"))
	case "slice":
		return fmt.Sprintf("encSlice(%s, e -> %s)", v, javaEncode(t.Elem, "e"))
	case "array":
		return fmt.Sprintf("encArray(%s, e -> %s)", v, javaEncode(t.Elem, "e"))
	case "map":
		return fmt.Sprintf("encMap(%s, e -> %s, e -> %s)", v, javaEncode(t.Key, "e"), javaEncode(t.Elem, "e"))
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

func javaHarness(cat *contracts.Catalog, t *contracts.Target) ([]byte, error) {
	cases, err := collectCases(cat, t)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "// %s\nimport java.util.Arrays;\nimport java.util.HashMap;\nimport java.util.List;\nimport java.util.Map;\nimport rt.*;\n\n", Marker)
	b.WriteString("@SuppressWarnings({\"unchecked\", \"unused\"})\nfinal class HarnessGen extends Codec {\n")
	var names []string
	for n, ci := range cases {
		fn := fmt.Sprintf("case_%d", n)
		names = append(names, fmt.Sprintf("        CASES.put(%q, HarnessGen::%s);\n", ci.ID(), fn))
		fmt.Fprintf(&b, "    static List<Object> %s(Harness.H h) {\n", fn)
		for _, l := range ci.lets {
			name := "v_" + l.name
			switch l.kind {
			case "view":
				fmt.Fprintf(&b, "        var %s = view(v_%s, h.let(%q));\n", name, l.of, l.name)
			case "ref":
				fmt.Fprintf(&b, "        var %s = v_%s;\n", name, l.of)
			default:
				fmt.Fprintf(&b, "        var %s = %s;\n", name, javaDecode(l.t, fmt.Sprintf("h.let(%q)", l.name)))
			}
		}
		var args []string
		for _, a := range ci.c.Call {
			args = append(args, "v_"+a)
		}
		vars := map[string]string{}
		for k, v := range ci.args {
			vars[k+".kind"] = tsKind(v)
			vars[k+".zero"] = javaZero(v)
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
		case strings.HasPrefix(call, "await:"):
			fmt.Fprintf(&b, "        Object[] rv = TaskSpawn.runIsolated(t -> { %s; });\n", strings.TrimSpace(strings.TrimPrefix(call, "await:")))
			for i := range ci.outs {
				fmt.Fprintf(&b, "        var r%d = rv[%d];\n", i, i)
			}
		case len(ci.outs) == 0:
			fmt.Fprintf(&b, "        %s;\n", call)
		case len(ci.outs) == 1:
			fmt.Fprintf(&b, "        var r0 = %s;\n", call)
		default:
			fmt.Fprintf(&b, "        Object[] rv = (Object[]) (%s);\n", call)
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
			fmt.Fprintf(&b, "        h.after(%q, %s);\n", a.Name, javaEncode(lt, "v_"+a.Name))
		}
		var encs []string
		for i, o := range ci.outs {
			encs = append(encs, javaEncode(o, fmt.Sprintf("r%d", i)))
		}
		fmt.Fprintf(&b, "        return Arrays.asList(new Object[] {%s});\n    }\n\n", join(encs))
	}
	b.WriteString("    static final Map<String, Harness.Case> CASES = new HashMap<>();\n\n    static {\n")
	for _, n := range names {
		b.WriteString(n)
	}
	b.WriteString("    }\n}\n")
	return b.Bytes(), nil
}
