package driver

import (
	"goalchemy/internal/diagnostics"
	"goalchemy/internal/emit/golang"
	"goalchemy/internal/link"
)

func init() {
	Register("go", emitGo)
}

func emitErr(code, msg string) []diagnostics.Diagnostic {
	return []diagnostics.Diagnostic{{Code: code, Severity: diagnostics.Error, Feature: "emission", Message: msg,
		Remedy: "This is a compiler defect; please report it with the source program."}}
}

func emitGo(res *Result, out string) []diagnostics.Diagnostic {
	o, err := golang.Emit(res.IR)
	if err != nil {
		_ = link.WriteFile(out, "main.go", o.Source)
		return emitErr("GCE004", err.Error())
	}
	refs, files, ds := link.Plan(res.Catalog, "go", o.Contracts)
	if len(ds) > 0 {
		return ds
	}
	rtFiles, err := link.CopyRuntime(res.Catalog, "go", files, out, "rt", true)
	if err != nil {
		return emitErr("GCE005", err.Error())
	}
	if err := link.WriteFile(out, "main.go", o.Source); err != nil {
		return emitErr("GCE005", err.Error())
	}
	if err := link.WriteFile(out, "go.mod", []byte("module goalchemyout\n\ngo 1.25\n")); err != nil {
		return emitErr("GCE005", err.Error())
	}
	if err := link.WriteManifest(out, res.Catalog, "go", refs, rtFiles, []string{"main.go", "go.mod"}, res.Program); err != nil {
		return emitErr("GCE005", err.Error())
	}
	return nil
}
