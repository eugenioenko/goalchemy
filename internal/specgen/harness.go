package specgen

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/eugenioenko/goalchemy/internal/contracts"
)

// caseInfo carries a case with its instantiated types for generators.
type caseInfo struct {
	fn     *contracts.Function
	impl   contracts.Implementation
	c      contracts.Case
	args   map[string]*contracts.TypeExpr
	lets   []letInfo
	inputs []*contracts.TypeExpr
	outs   []*contracts.TypeExpr
}

type letInfo struct {
	name string
	t    *contracts.TypeExpr
	kind string // value, view, ref
	of   string // view or ref target
}

func (ci *caseInfo) ID() string { return ci.fn.ID + "/" + ci.c.Name }

// collectCases returns every case of every implemented function, sorted.
func collectCases(cat *contracts.Catalog, t *contracts.Target) ([]*caseInfo, error) {
	var out []*caseInfo
	for _, impl := range t.Functions {
		fc := cat.Functions[impl.ID]
		if impl.Harness == "" {
			continue
		}
		for _, c := range fc.Cases {
			ci := &caseInfo{fn: fc, impl: impl, c: c, args: map[string]*contracts.TypeExpr{}}
			for k, v := range c.Types {
				te, err := contracts.ParseType(v)
				if err != nil {
					return nil, err
				}
				ci.args[k] = te
			}
			for _, b := range c.Let {
				te, err := contracts.ParseType(b.Type)
				if err != nil {
					return nil, err
				}
				li := letInfo{name: b.Name, t: te.Substitute(ci.args), kind: "value"}
				var raw map[string]any
				if json.Unmarshal(b.Value, &raw) == nil {
					if v, ok := raw["view"].(string); ok {
						li.kind, li.of = "view", v
					} else if v, ok := raw["ref"].(string); ok {
						li.kind, li.of = "ref", v
					}
				}
				ci.lets = append(ci.lets, li)
			}
			for _, p := range fc.Signature.Inputs {
				te, _ := contracts.ParseType(p.Type)
				ci.inputs = append(ci.inputs, te.Substitute(ci.args))
			}
			for _, p := range fc.Signature.Outputs {
				te, _ := contracts.ParseType(p.Type)
				ci.outs = append(ci.outs, te.Substitute(ci.args))
			}
			out = append(out, ci)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out, nil
}

// expand substitutes {N} with args and {T} with rendered type arguments.
func expand(tmpl string, args []string, types map[string]string) string {
	s := tmpl
	for i := len(args) - 1; i >= 0; i-- {
		s = strings.ReplaceAll(s, fmt.Sprintf("{%d}", i), args[i])
	}
	for k, v := range types {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}
