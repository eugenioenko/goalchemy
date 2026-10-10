package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/testutil"
)

// The Node package entry installs the file-system adapter; the browser entry
// does not, so std/os must fail with an unsupported error there.
func TestGeneratedTypeScriptFiles(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	run := func(dir, name string, args ...string) string {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir, cmd.Env = dir, driver.ToolEnv()
		data, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", name, err, data)
		}
		return string(data)
	}
	write := func(dir, name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	source, out := t.TempDir(), t.TempDir()
	write(source, "go.mod", fmt.Sprintf("module filesprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root))
	write(source, "library.go", `package filesprobe
import ("github.com/eugenioenko/goalchemy/lib/context";"github.com/eugenioenko/goalchemy/std/os")
func Read(ctx context.Context, name string) (string, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return "error: " + err.Error(), nil
	}
	return string(data), nil
}
func Write(ctx context.Context, name, text string) (string, error) {
	if err := os.WriteFile(name, []byte(text), 0o600); err != nil {
		return "error: " + err.Error(), nil
	}
	return "ok", nil
}
`)
	if ds := testutil.CompileGate(source, "typescript", out, "cooperative"); len(ds) > 0 {
		t.Fatal(ds)
	}
	run(out, "tsc", "-p", ".")
	write(out, "consumer.mjs", `import assert from 'node:assert/strict';
const g=await import('goalchemy-generated');
assert.equal(await g.Write('note.txt','hello'),'ok');
assert.equal(await g.Read('note.txt'),'hello');
assert.equal(await g.Read('missing.txt'),'error: open missing.txt: no such file or directory');
console.log('PASS generated Node package file system');`)
	t.Log(run(out, "node", "consumer.mjs"))
	t.Log(run(root, "node", "targets/typescript/tests/files_browser.mjs", out))
}
