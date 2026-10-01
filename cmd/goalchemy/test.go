package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"goalchemy/internal/catalog"
	"goalchemy/internal/conformance"
	"goalchemy/internal/diagnostics"
)

func runTest(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".", "repository root containing targets and their harnesses")
	verbose := fs.Bool("v", false, "list every case")
	var targets multiFlag
	fs.Var(&targets, "target", "target to test (repeatable; default: all with harnesses)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cat, ds := catalog.Load(os.DirFS(*root))
	if diagnostics.HasErrors(ds) {
		_ = diagnostics.Write(stderr, ds, false)
		return 1
	}
	if len(targets) == 0 {
		for n, t := range cat.Targets {
			if t.Harness != nil {
				targets = append(targets, n)
			}
		}
		sort.Strings(targets)
	}
	failed := 0
	for _, t := range targets {
		results, err := conformance.Run(cat, *root, t)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", t, err)
			failed++
			continue
		}
		pass := 0
		for _, r := range results {
			if r.Pass {
				pass++
				if *verbose {
					fmt.Fprintf(stdout, "ok   %s %s\n", t, r.Case)
				}
			} else {
				failed++
				fmt.Fprintf(stdout, "FAIL %s %s: %s\n", t, r.Case, r.Detail)
			}
		}
		fmt.Fprintf(stdout, "%s: %d/%d contract cases passed\n", t, pass, len(results))
	}
	if failed > 0 {
		return 1
	}
	return 0
}

type multiFlag []string

func (m *multiFlag) String() string     { return fmt.Sprint(*m) }
func (m *multiFlag) Set(s string) error { *m = append(*m, s); return nil }
