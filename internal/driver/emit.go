package driver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/eugenioenko/goalchemy/internal/diagnostics"
	"github.com/eugenioenko/goalchemy/internal/ir"
	"github.com/eugenioenko/goalchemy/internal/link"
)

// Emitter writes a complete target build directory.
type Emitter func(res *Result, out string) []diagnostics.Diagnostic

var emitters = map[string]Emitter{}

func Register(name string, e Emitter) { emitters[name] = e }

func Targets() []string {
	var out []string
	for n := range emitters {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func Emit(name string, res *Result, out string) (ds []diagnostics.Diagnostic) {
	defer func() {
		if r := recover(); r != nil {
			ds = []diagnostics.Diagnostic{{Code: "GCE004", Severity: diagnostics.Error, Feature: "emission",
				Message: fmt.Sprintf("%s emitter failed: %v", name, r),
				Remedy:  "This is a compiler defect; please report it with the source program."}}
		}
	}()
	e, ok := emitters[name]
	if !ok {
		return []diagnostics.Diagnostic{{Code: "GCE001", Severity: diagnostics.Error, Feature: "target",
			Message: "unknown or unimplemented target " + name, Remedy: "Choose one of: " + joinTargets()}}
	}
	if res.IR != nil && res.IR.Library && name != "c" && name != "ir" {
		return []diagnostics.Diagnostic{{Code: "GCE006", Severity: diagnostics.Error, Feature: "library build",
			Message: "target " + name + " cannot build a library; only the c target exports a library API",
			Remedy:  "Compile package main for this target, or use -target c for a library."}}
	}
	// Fail before target type lowering when a declared capability has no
	// implementation; opaque native keys must never become fake target values.
	if res.IR != nil && res.Catalog != nil && name != "ir" {
		var used []string
		for _, external := range res.IR.Externals {
			used = append(used, external.Contract)
		}
		// Type-only opaque handles also need a runtime representation. Use
		// the first declaration-backed receiver contract as its owner.
		ids := res.Catalog.IDs()
		if res.IR.Types != nil {
			for _, typ := range res.IR.Types.All {
				if typ.Kind != ir.KOpaque {
					continue
				}
				prefix := typ.Pkg + ".(" + typ.Obj + ")."
				for _, id := range ids {
					fn := res.Catalog.Functions[id]
					if fn != nil && strings.HasPrefix(fn.Signature.Symbol, prefix) {
						used = append(used, id)
						break
					}
				}
			}
		}
		sort.Strings(used)
		_, _, ds := link.Plan(res.Catalog, name, used)
		if diagnostics.HasErrors(ds) {
			return ds
		}
	}
	return e(res, out)
}

func joinTargets() string {
	s := ""
	for i, t := range Targets() {
		if i > 0 {
			s += ", "
		}
		s += t
	}
	return s
}
