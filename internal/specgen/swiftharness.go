package specgen

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/eugenioenko/goalchemy/internal/contracts"
)

func init() { harnessGenerators["swift"] = swiftHarness }

func swiftHarness(cat *contracts.Catalog, t *contracts.Target) ([]byte, error) {
	cases, err := collectCases(cat, t)
	if err != nil {
		return nil, err
	}
	types := map[string]*contracts.TypeExpr{}
	var add func(*contracts.TypeExpr)
	add = func(t *contracts.TypeExpr) {
		if t == nil {
			return
		}
		if _, ok := types[t.String()]; ok {
			return
		}
		types[t.String()] = t
		add(t.Elem)
		add(t.Key)
	}
	for _, ci := range cases {
		for _, l := range ci.lets {
			add(l.t)
		}
		for _, o := range ci.outs {
			add(o)
		}
		for _, a := range ci.args {
			add(a)
		}
	}
	for _, s := range []string{"uint8", "string", "struct{}", "[]rune", "[2]int"} {
		t, _ := contracts.ParseType(s)
		add(t)
	}
	var order []string
	for s := range types {
		order = append(order, s)
	}
	sort.Strings(order)
	ids := map[string]int{}
	for i, s := range order {
		ids[s] = i
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "// %s\nimport Foundation\n\nfunc gHarnessInitialize() {\n    GTypes.table = [\n", Marker)
	for _, s := range order {
		te := types[s]
		kind := te.Kind
		bits := uint(0)
		signed := false
		elem, key := -1, -1
		if te.Elem != nil {
			elem = ids[te.Elem.String()]
		}
		if te.Key != nil {
			key = ids[te.Key.String()]
		}
		if te.Kind == "name" {
			if w, sign, ok := contracts.IntInfo(te.Name); ok {
				kind, bits, signed = "int", w, sign
			} else {
				switch te.Name {
				case "bool":
					kind = "bool"
				case "string":
					kind = "string"
				case "float32":
					kind, bits = "float", 32
				case "float64":
					kind, bits = "float", 64
				case "error", "any":
					kind = "interface"
				case "struct{}":
					kind = "struct"
				case "func()", "context.CancelFunc":
					kind = "func"
				default:
					kind = "opaque"
				}
			}
		}
		fmt.Fprintf(&b, "        GType(%q, %q, %d, %t, %d, %d, %d, [], [], [], [], %t),\n", s, kind, bits, signed, elem, key, te.Len, te.Comparable())
	}
	b.WriteString("    ]\n}\n\nlet gHarnessCases: [String: GHarnessCase] = [\n")
	for _, ci := range cases {
		var bindings, args, out, params []string
		for _, l := range ci.lets {
			bindings = append(bindings, fmt.Sprintf("GHarnessBinding(%q, %d, %q, %q)", l.name, ids[l.t.String()], l.kind, l.of))
		}
		for _, a := range ci.c.Call {
			args = append(args, quote(a))
		}
		for _, o := range ci.outs {
			out = append(out, fmt.Sprint(ids[o.String()]))
		}
		var ps []string
		for name := range ci.args {
			ps = append(ps, name)
		}
		sort.Strings(ps)
		for _, p := range ps {
			params = append(params, fmt.Sprintf("%q: %d", p, ids[ci.args[p].String()]))
		}
		generic := "[:]"
		if len(params) > 0 {
			generic = "[" + strings.Join(params, ", ") + "]"
		}
		fmt.Fprintf(&b, "    %q: GHarnessCase(%q, [%s], [%s], [%s], %s, %s),\n", ci.ID(), ci.fn.ID, strings.Join(bindings, ", "), strings.Join(args, ", "), strings.Join(out, ", "), generic, ci.impl.Symbol)
	}
	b.WriteString("]\n")
	return b.Bytes(), nil
}
