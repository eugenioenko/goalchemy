package driver

import (
	"fmt"
	"sort"

	"github.com/eugenioenko/goalchemy/internal/diagnostics"
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
