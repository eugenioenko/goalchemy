package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/eugenioenko/goalchemy/internal/catalog"
	"github.com/eugenioenko/goalchemy/internal/diagnostics"
	"github.com/eugenioenko/goalchemy/internal/link"
	"github.com/eugenioenko/goalchemy/internal/specgen"
)

func runSpec(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: goalchemy spec validate [-json] [-root dir]")
		return 2
	}
	switch args[0] {
	case "validate":
		fs := flag.NewFlagSet("spec validate", flag.ContinueOnError)
		fs.SetOutput(stderr)
		asJSON := fs.Bool("json", false, "write diagnostics as JSON")
		root := fs.String("root", "", "catalog root directory (default: embedded catalog)")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		fsys := catalog.FS()
		if *root != "" {
			fsys = os.DirFS(*root)
		}
		cat, ds := catalog.Load(fsys)
		if *asJSON {
			_ = diagnostics.Write(stdout, ds, true)
		} else {
			_ = diagnostics.Write(stderr, ds, false)
			if !diagnostics.HasErrors(ds) {
				fmt.Fprintf(stdout, "ok: %d type contracts, %d function contracts, %d targets\n",
					len(cat.Types), len(cat.Functions), len(cat.Targets))
			}
		}
		if diagnostics.HasErrors(ds) {
			return 1
		}
		return 0
	case "generate":
		fs := flag.NewFlagSet("spec generate", flag.ContinueOnError)
		fs.SetOutput(stderr)
		check := fs.Bool("check", false, "verify generated files are current instead of writing them")
		root := fs.String("root", ".", "repository root")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		cat, ds := catalog.Load(os.DirFS(*root))
		if diagnostics.HasErrors(ds) {
			_ = diagnostics.Write(stderr, ds, false)
			return 1
		}
		files, err := specgen.Generate(cat)
		if err != nil {
			fmt.Fprintln(stderr, "spec generate:", err)
			return 1
		}
		if *check {
			problems := specgen.Check(cat, files)
			for _, p := range problems {
				fmt.Fprintln(stderr, "GCC009:", p)
			}
			if len(problems) > 0 {
				fmt.Fprintln(stderr, "run goalchemy spec generate to refresh generated files")
				return 1
			}
			fmt.Fprintf(stdout, "ok: %d generated files are current\n", len(files))
			return 0
		}
		for p, data := range files {
			if err := link.WriteFile(*root, p, data); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
		}
		fmt.Fprintf(stdout, "wrote %d generated files\n", len(files))
		return 0
	}
	fmt.Fprintf(stderr, "goalchemy spec: unknown subcommand %q\n", args[0])
	return 2
}
