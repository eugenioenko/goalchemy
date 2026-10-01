// Package language runs the source language fixtures: each program runs
// natively and through every released target, and observations must agree.
package language

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"goalchemy/internal/testutil"
)

var targets = []string{"go"}

func TestFixtures(t *testing.T) {
	fixtures, err := testutil.Discover("testdata")
	if err != nil {
		t.Fatal(err)
	}
	if only := os.Getenv("FIXTURE"); only != "" {
		var keep []testutil.Fixture
		for _, f := range fixtures {
			if strings.Contains(f.Name, only) {
				keep = append(keep, f)
			}
		}
		fixtures = keep
	}
	for _, f := range fixtures {
		t.Run(f.Name, func(t *testing.T) {
			t.Parallel()
			work := t.TempDir()
			if len(f.Reject) > 0 {
				ds := testutil.Compile(f.Dir, "go", filepath.Join(work, "go"))
				got := map[string]bool{}
				for _, d := range ds {
					got[d.Code] = true
				}
				for _, c := range f.Reject {
					if !got[c] {
						t.Errorf("expected diagnostic %s, got %v", c, ds)
					}
				}
				return
			}
			native, err := testutil.Native(f.Dir, work)
			if err != nil {
				t.Fatal(err)
			}
			want := testutil.Normalize(native)
			var first *testutil.Observation
			for _, target := range targets {
				out := filepath.Join(work, target)
				if ds := testutil.Compile(f.Dir, target, out); len(ds) > 0 {
					for _, d := range ds {
						t.Errorf("%s: %s", target, d)
					}
					continue
				}
				obs, err := testutil.Runners[target](out)
				if err != nil {
					t.Errorf("%s: %v", target, err)
					continue
				}
				got := testutil.Normalize(obs)
				cmpWant, cmpGot := want, got
				if f.Unordered {
					cmpWant, cmpGot = testutil.Unordered(want), testutil.Unordered(got)
				}
				if cmpGot != cmpWant {
					t.Errorf("%s differs from native Go\n=== native\n%s=== %s\n%s", target, want, target, got)
				}
				if first == nil {
					first = &got
				} else if got != *first {
					t.Errorf("%s differs from %s\n=== %s\n%s=== %s\n%s", target, targets[0], targets[0], *first, target, got)
				}
			}
		})
	}
}
