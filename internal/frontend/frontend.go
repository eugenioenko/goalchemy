// Package frontend loads Go packages under the locked Goalchemy source profile.
package frontend

import (
	"context"
	"fmt"
	"go/token"
	"go/version"
	"os"
	"sort"
	"strconv"
	"strings"

	"goalchemy/internal/diagnostics"

	"golang.org/x/tools/go/packages"
)

const (
	SourceLanguage     = "go1.25"
	ReferenceToolchain = "go1.27.1"
	SourceGOOS         = "linux"
	SourceGOARCH       = "amd64"
)

type Options struct {
	Dir      string
	Patterns []string
	Tags     []string
	// External reports whether an import path is a declared external capability
	// package rather than included source.
	External func(path string) bool
}

type Program struct {
	Fset *token.FileSet
	// Roots are the packages matched by the patterns.
	Roots []*packages.Package
	// Source holds every included source package in dependency order.
	Source []*packages.Package
	// External holds imported packages that are not included source.
	External []*packages.Package
	Tags     []string
}

func (p *Program) IsSource(path string) bool {
	for _, s := range p.Source {
		if s.PkgPath == path {
			return true
		}
	}
	return false
}

func Load(ctx context.Context, opts Options) (*Program, []diagnostics.Diagnostic) {
	patterns := opts.Patterns
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	fset := token.NewFileSet()
	env := append(os.Environ(),
		"GOOS="+SourceGOOS, "GOARCH="+SourceGOARCH, "CGO_ENABLED=0",
		"GOTOOLCHAIN="+ReferenceToolchain, "GOFLAGS=-mod=mod")
	cfg := &packages.Config{
		Context: ctx,
		Dir:     opts.Dir,
		Fset:    fset,
		Env:     env,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedModule |
			packages.NeedImports | packages.NeedDeps,
	}
	tags := append([]string(nil), opts.Tags...)
	sort.Strings(tags)
	if len(tags) > 0 {
		cfg.BuildFlags = []string{"-tags=" + strings.Join(tags, ",")}
	}
	prog := &Program{Fset: fset, Tags: tags}
	var ds []diagnostics.Diagnostic
	roots, err := packages.Load(cfg, patterns...)
	if err != nil {
		ds = append(ds, diagnostics.Diagnostic{Code: "GCL001", Severity: diagnostics.Error,
			Feature: "package loading", Message: err.Error(),
			Remedy: "Check the package patterns and the Go module configuration."})
		return prog, ds
	}
	if len(roots) == 0 {
		ds = append(ds, diagnostics.Diagnostic{Code: "GCL002", Severity: diagnostics.Error,
			Feature: "source selection", Message: "no packages matched " + strings.Join(patterns, " "),
			Remedy: "Provide a package pattern that matches Go source files."})
		return prog, ds
	}
	prog.Roots = roots
	external := opts.External
	if external == nil {
		external = func(string) bool { return false }
	}
	seen := map[string]bool{}
	var visit func(p *packages.Package)
	visit = func(p *packages.Package) {
		if seen[p.PkgPath] {
			return
		}
		seen[p.PkgPath] = true
		if external(p.PkgPath) || !isSource(p) {
			prog.External = append(prog.External, p)
			return
		}
		paths := make([]string, 0, len(p.Imports))
		for path := range p.Imports {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			visit(p.Imports[path])
		}
		prog.Source = append(prog.Source, p)
	}
	for _, r := range roots {
		visit(r)
	}
	for _, p := range prog.Source {
		ds = append(ds, packageErrors(p)...)
		ds = append(ds, languageVersion(fset, p)...)
	}
	for _, r := range roots {
		if !prog.IsSource(r.PkgPath) {
			ds = append(ds, diagnostics.Diagnostic{Code: "GCL002", Severity: diagnostics.Error,
				Symbol: r.PkgPath, Feature: "source selection",
				Message: fmt.Sprintf("root package %s is not part of the main module", r.PkgPath),
				Remedy:  "Select packages from the module being transpiled."})
		}
	}
	diagnostics.Sort(ds)
	return prog, diagnostics.Dedup(ds)
}

func isSource(p *packages.Package) bool {
	return p.Module != nil && p.Module.Main
}

func packageErrors(p *packages.Package) []diagnostics.Diagnostic {
	var ds []diagnostics.Diagnostic
	for _, e := range p.Errors {
		code, feature := "GCL003", "Go parsing or typing"
		remedy := "Fix the Go error; Goalchemy analyzes only valid Go."
		if e.Kind == packages.ListError {
			code, feature = "GCL001", "package loading"
			remedy = "Fix the package configuration or import path."
		}
		file, line, col := parsePosition(e.Pos)
		ds = append(ds, diagnostics.Diagnostic{Code: code, Severity: diagnostics.Error,
			File: file, Line: line, Column: col, Symbol: p.PkgPath, Feature: feature,
			Message: e.Msg, Remedy: remedy})
	}
	if len(p.Errors) > 0 {
		ds = append(ds, diagnostics.Diagnostic{Code: "GCL005", Severity: diagnostics.Error,
			Symbol: p.PkgPath, Feature: "analysis limit",
			Message: "package " + p.PkgPath + " has Go errors; subset findings for it may be incomplete",
			Remedy:  "Fix the Go errors and run the check again."})
	}
	return ds
}

func languageVersion(fset *token.FileSet, p *packages.Package) []diagnostics.Diagnostic {
	var ds []diagnostics.Diagnostic
	tooNew := func(v string) bool {
		v = version.Lang(v)
		return v != "" && version.Compare(v, SourceLanguage) > 0
	}
	if p.Module != nil && p.Module.GoVersion != "" && tooNew("go"+p.Module.GoVersion) {
		ds = append(ds, diagnostics.Diagnostic{Code: "GCL004", Severity: diagnostics.Error,
			File: p.Module.GoMod, Symbol: p.PkgPath, Feature: "source language version",
			Message: fmt.Sprintf("module requires go %s, newer than the Goalchemy baseline %s", p.Module.GoVersion, SourceLanguage),
			Remedy:  "Set the module go directive to 1.25 or earlier."})
	}
	if p.TypesInfo != nil {
		for _, f := range p.Syntax {
			if v := p.TypesInfo.FileVersions[f]; v != "" && tooNew(v) {
				d := diagnostics.Span(fset, f.Package, token.NoPos)
				d.Code, d.Severity, d.Symbol = "GCL004", diagnostics.Error, p.PkgPath
				d.Feature = "source language version"
				d.Message = fmt.Sprintf("file selects language version %s, newer than %s", v, SourceLanguage)
				d.Remedy = "Remove the go version build constraint or lower it to go1.25."
				ds = append(ds, d)
			}
		}
	}
	return ds
}

func parsePosition(s string) (string, int, int) {
	parts := strings.Split(s, ":")
	if len(parts) >= 3 {
		line, err1 := strconv.Atoi(parts[len(parts)-2])
		col, err2 := strconv.Atoi(parts[len(parts)-1])
		if err1 == nil && err2 == nil {
			return strings.Join(parts[:len(parts)-2], ":"), line, col
		}
	}
	if len(parts) >= 2 {
		if line, err := strconv.Atoi(parts[len(parts)-1]); err == nil {
			return strings.Join(parts[:len(parts)-1], ":"), line, 0
		}
	}
	return s, 0, 0
}
