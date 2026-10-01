// Package contracts checks generated files and runs every canonical
// contract case through each target harness.
package contracts

import (
	"os"
	"testing"

	"goalchemy/internal/catalog"
	"goalchemy/internal/conformance"
	"goalchemy/internal/specgen"
)

const root = "../.."

func TestGeneratedFilesAreCurrent(t *testing.T) {
	cat, ds := catalog.Load(os.DirFS(root))
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	files, err := specgen.Generate(cat)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range specgen.Check(cat, files) {
		t.Error(p)
	}
}

func TestTargetConformance(t *testing.T) {
	cat, ds := catalog.Load(os.DirFS(root))
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	for name, tgt := range cat.Targets {
		if tgt.Harness == nil {
			continue
		}
		t.Run(name, func(t *testing.T) {
			results, err := conformance.Run(cat, root, name)
			if err != nil {
				t.Fatal(err)
			}
			for _, r := range results {
				if !r.Pass {
					t.Errorf("%s: %s", r.Case, r.Detail)
				}
			}
		})
	}
}
