package driver

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/link"
)

func TestFormatRewritesSourcesAndDropsPositionMaps(t *testing.T) {
	if _, err := exec.LookPath("gofmt"); err != nil {
		t.Skip("gofmt is unavailable")
	}
	out := t.TempDir()
	files := map[string]string{
		"main.go":       "package main\nfunc main( ) {println(1)}\n",
		"main.go.lines": "1 a.go:1:1\n",
		"main.ts":       "x\n//# sourceMappingURL=main.ts.map\n",
		"main.ts.map":   "{}",
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(out, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := link.Manifest{GeneratedFiles: []string{"main.go", "main.go.lines", "main.ts", "main.ts.map"}}
	data, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(out, "goalchemy.manifest.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if ds := Format("go", out); len(ds) != 0 {
		t.Fatal(ds)
	}
	src, _ := os.ReadFile(filepath.Join(out, "main.go"))
	if !strings.Contains(string(src), "package main\n\nfunc main() { println(1) }") {
		t.Fatalf("not formatted:\n%s", src)
	}
	for _, gone := range []string{"main.go.lines", "main.ts.map"} {
		if _, err := os.Stat(filepath.Join(out, gone)); !os.IsNotExist(err) {
			t.Fatal(gone, "was kept")
		}
	}
	ts, _ := os.ReadFile(filepath.Join(out, "main.ts"))
	if strings.Contains(string(ts), "sourceMappingURL") {
		t.Fatal("source map comment was kept")
	}
	raw, _ := os.ReadFile(filepath.Join(out, "goalchemy.manifest.json"))
	var got link.Manifest
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.GeneratedFiles, ",") != "main.go,main.ts" {
		t.Fatal(got.GeneratedFiles)
	}
}

func TestFormatWarnsWhenFormatterIsMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Chdir(t.TempDir())
	ds := Format("java", t.TempDir())
	if len(ds) != 1 || ds[0].Code != "GCE008" || ds[0].Severity != "warning" {
		t.Fatal(ds)
	}
}
