// Package integration covers the compiler driver end to end: reproducible
// output, manifests, and diagnostics for rejected programs.
package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/link"
	"github.com/eugenioenko/goalchemy/internal/testutil"
)

var fixtures = []string{"../language/testdata/calls_results", "../language/testdata/globals_init"}

func readTree(t *testing.T, dir string) map[string][]byte {
	out := map[string][]byte{}
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		out[rel], err = os.ReadFile(p)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestReproducibleOutput(t *testing.T) {
	targets := []string{"go", "typescript", "python", "java", "csharp", "rust", "c", "swift"}
	if testing.Short() {
		targets = targets[:2]
	}
	builds := map[string][2]*driver.Result{}
	// Preserve independent catalog and source loads for both runs.
	first, second := testutil.NewSuite(t), testutil.NewSuite(t)
	for _, f := range fixtures {
		var pair [2]*driver.Result
		for i, suite := range []*testutil.Suite{first, second} {
			res, ds := suite.Build(testutil.Fixture{Dir: f})
			if len(ds) > 0 {
				t.Fatal(ds)
			}
			pair[i] = res
		}
		builds[f] = pair
	}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			for _, f := range fixtures {
				a, b := t.TempDir(), t.TempDir()
				if ds := driver.EmitWithOptions(target, builds[f][0], a, driver.EmitOptions{}); len(ds) > 0 {
					t.Fatal(ds)
				}
				if ds := driver.EmitWithOptions(target, builds[f][1], b, driver.EmitOptions{}); len(ds) > 0 {
					t.Fatal(ds)
				}
				ta, tb := readTree(t, a), readTree(t, b)
				if len(ta) != len(tb) {
					t.Fatalf("%s %s: file sets differ", target, f)
				}
				for p, data := range ta {
					if !bytes.Equal(data, tb[p]) {
						t.Errorf("%s %s: %s differs between runs", target, f, p)
					}
				}
			}
		})
	}
}

func TestManifest(t *testing.T) {
	out := t.TempDir()
	if ds := testutil.Compile("../language/testdata/calls_results", "go", out); len(ds) > 0 {
		t.Fatal(ds)
	}
	data, err := os.ReadFile(filepath.Join(out, "goalchemy.manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m link.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m.Target != "go" || m.SourceProfile["language"] != "go1.25" || len(m.RuntimeFiles) == 0 {
		t.Errorf("unexpected manifest %+v", m)
	}
	for _, rf := range m.RuntimeFiles {
		if _, err := os.Stat(filepath.Join(out, rf)); err != nil {
			t.Errorf("manifest lists missing runtime file %s", rf)
		}
	}
}

func TestRejectedProgram(t *testing.T) {
	ds := testutil.Compile("../../internal/subset/testdata/reject_stmts", "go", t.TempDir())
	if len(ds) == 0 {
		t.Fatal("expected diagnostics")
	}
	for _, d := range ds {
		if d.Code == "" || d.Remedy == "" || d.File == "" {
			t.Errorf("incomplete diagnostic %+v", d)
		}
	}
}
