// Package clib builds a Goalchemy library for the C target and calls it
// from a small C host program.
package clib

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/testutil"
)

func TestCHost(t *testing.T) {
	out := t.TempDir()
	if ds := testutil.Compile("testdata/calc", "c", out); len(ds) > 0 {
		t.Fatal(ds)
	}
	for _, f := range []string{"goalchemy.h", "build.sh"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Fatalf("library output lacks %s", f)
		}
	}
	env := driver.ToolEnv()
	sh := func(dir string, name string, args ...string) string {
		cmd := exec.Command(name, args...)
		cmd.Dir, cmd.Env = dir, env
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, b)
		}
		return string(b)
	}
	sh(out, "sh", "build.sh")
	host, _ := filepath.Abs("testdata/host.c")
	gc := "-lgc"
	link := []string{"-std=c17", "-I.", host, "libgoalchemy.a"}
	for _, kv := range env {
		if len(kv) > len("GOALCHEMY_BDWGC=") && kv[:len("GOALCHEMY_BDWGC=")] == "GOALCHEMY_BDWGC=" {
			gc = filepath.Join(kv[len("GOALCHEMY_BDWGC="):], "lib", "libgc.a")
		}
	}
	link = append(link, gc, "-lpthread", "-o", "host")
	sh(out, "cc", link...)
	got := sh(out, filepath.Join(out, "host"))
	want, _ := os.ReadFile("testdata/want.txt")
	if got != string(want) {
		t.Errorf("host output differs\n=== want\n%s=== got\n%s", want, got)
	}
}
