package integration

import (
	"context"
	"fmt"
	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/testutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGeneratedPythonImportingLibrary(t *testing.T) {
	root, e := filepath.Abs("../..")
	if e != nil {
		t.Fatal(e)
	}
	source, parent := t.TempDir(), t.TempDir()
	out := filepath.Join(parent, "probe")
	fixture, e := os.ReadFile(filepath.Join(root, "targets/python/tests/library_fixture.go.in"))
	if e != nil {
		t.Fatal(e)
	}
	mod := fmt.Sprintf("module pythonlibraryprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root)
	for name, data := range map[string][]byte{"go.mod": []byte(mod), "library.go": fixture} {
		if e := os.WriteFile(filepath.Join(source, name), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	if ds := testutil.CompileGate(source, "python", out, "cooperative"); len(ds) > 0 {
		t.Fatal(ds)
	}
	generated, e := os.ReadFile(filepath.Join(out, "main.py"))
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(generated), "from . import rt") || !strings.Contains(string(generated), "rt.library_submit(") {
		t.Fatal("missing independent package exports")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", filepath.Join(root, "targets/python/tests/library_test.py.in"), parent)
	cmd.Env = driver.ToolEnv()
	if data, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("actual importing CPython consumer: %v\n%s", e, data)
	} else {
		t.Log(string(data))
	}
}
