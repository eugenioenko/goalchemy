package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"goalchemy/internal/catalog"
	"goalchemy/internal/diagnostics"
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
	}
	fmt.Fprintf(stderr, "goalchemy spec: unknown subcommand %q\n", args[0])
	return 2
}
