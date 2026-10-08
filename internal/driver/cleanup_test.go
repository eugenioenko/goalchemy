package driver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/contracts"
	"github.com/eugenioenko/goalchemy/internal/diagnostics"
	"github.com/eugenioenko/goalchemy/internal/ir"
	"github.com/eugenioenko/goalchemy/internal/link"
)

func cleanupInventory(t *testing.T, out string, generated, runtime []string) {
	t.Helper()
	data, err := json.Marshal(link.Manifest{GeneratedFiles: generated, RuntimeFiles: runtime})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "goalchemy.manifest.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}
func cleanupFile(t *testing.T, out, name string) {
	t.Helper()
	file := filepath.Join(out, name)
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(name), 0600); err != nil {
		t.Fatal(err)
	}
}
func cleanupExists(t *testing.T, out, name string, want bool) {
	t.Helper()
	_, err := os.Stat(filepath.Join(out, name))
	if (err == nil) != want || err != nil && !os.IsNotExist(err) {
		t.Fatalf("%s existence: %v; want %v", name, err, want)
	}
}

func TestSuccessfulEmissionCleansOnlyPreviousInventory(t *testing.T) {
	const target = "test-inventory-cleanup"
	out := t.TempDir()
	for _, file := range []string{"old.ts", "old.ts.map", "rt/unused.ts", "same.ts", "consumer.ts", "dist/caller.js"} {
		cleanupFile(t, out, file)
	}
	cleanupInventory(t, out, []string{"old.ts", "old.ts.map", "same.ts"}, []string{"rt/unused.ts"})
	Register(target, func(_ *Result, out string) []diagnostics.Diagnostic {
		cleanupFile(t, out, "new.ts")
		cleanupInventory(t, out, []string{"same.ts", "new.ts"}, nil)
		return nil
	})
	t.Cleanup(func() { delete(emitters, target) })
	if ds := Emit(target, &Result{}, out); diagnostics.HasErrors(ds) {
		t.Fatal(ds)
	}
	for _, file := range []string{"old.ts", "old.ts.map", "rt/unused.ts"} {
		cleanupExists(t, out, file, false)
	}
	for _, file := range []string{"same.ts", "new.ts", "consumer.ts", "dist/caller.js"} {
		cleanupExists(t, out, file, true)
	}
}

func TestFailedEmissionDoesNotCleanInventory(t *testing.T) {
	const target = "test-inventory-failure"
	for _, mode := range []string{"diagnostic", "panic", "preflight", "invalid-current"} {
		t.Run(mode, func(t *testing.T) {
			out := t.TempDir()
			cleanupFile(t, out, "old.ts")
			cleanupInventory(t, out, []string{"old.ts"}, nil)
			called := false
			Register(target, func(_ *Result, out string) []diagnostics.Diagnostic {
				called = true
				switch mode {
				case "panic":
					panic("emission failed")
				case "diagnostic":
					return emitErr("GCE004", "emission failed")
				case "invalid-current":
					cleanupInventory(t, out, []string{"../caller.ts"}, nil)
				}
				return nil
			})
			t.Cleanup(func() { delete(emitters, target) })
			res := &Result{}
			if mode == "preflight" {
				res.IR = &ir.Program{Externals: map[string]*ir.Extern{"bad": {Contract: "missing"}}}
				res.Catalog = &contracts.Catalog{}
			}
			if ds := Emit(target, res, out); !diagnostics.HasErrors(ds) {
				t.Fatal("missing failure")
			}
			if mode == "preflight" && called {
				t.Fatal("preflight entered emitter")
			}
			cleanupExists(t, out, "old.ts", true)
		})
	}
}

func TestInvalidManagedPathsPreventEmission(t *testing.T) {
	const target = "test-inventory-validation"
	for _, name := range []string{"../caller.ts", "/tmp/caller.ts", "a/../caller.ts", "a\\caller.ts", "C:caller.ts", ".", "goalchemy.manifest.json", "dir", "escape/caller.ts"} {
		t.Run(name, func(t *testing.T) {
			out := t.TempDir()
			cleanupFile(t, out, "old.ts")
			if err := os.Mkdir(filepath.Join(out, "dir"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(t.TempDir(), filepath.Join(out, "escape")); err != nil {
				t.Fatal(err)
			}
			cleanupInventory(t, out, []string{"old.ts", name}, nil)
			Register(target, func(_ *Result, _ string) []diagnostics.Diagnostic {
				t.Fatal("invalid inventory entered emitter")
				return nil
			})
			t.Cleanup(func() { delete(emitters, target) })
			if ds := Emit(target, &Result{}, out); !diagnostics.HasErrors(ds) {
				t.Fatal("invalid inventory accepted")
			}
			cleanupExists(t, out, "old.ts", true)
		})
	}
}

func TestIREmissionIgnoresOutputInventory(t *testing.T) {
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "goalchemy.manifest.json"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	// CLI registers IR separately from native targets. Its emission must not
	// inspect the inventory from an unrelated native build.
	old, exists := emitters["ir"]
	Register("ir", func(_ *Result, _ string) []diagnostics.Diagnostic { return nil })
	t.Cleanup(func() {
		if exists {
			emitters["ir"] = old
		} else {
			delete(emitters, "ir")
		}
	})
	if ds := Emit("ir", &Result{IR: &ir.Program{}}, out); diagnostics.HasErrors(ds) {
		t.Fatal(ds)
	}
}
