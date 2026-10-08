package driver

import (
	"encoding/json"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/emit/artifact"
	"github.com/eugenioenko/goalchemy/internal/sourcemap"
)

func TestSourceArtifactMapsUseNativeFileLocation(t *testing.T) {
	base := t.TempDir()
	out := filepath.Join(base, "out")
	source := filepath.Join(base, "source.go")
	m := &sourcemap.Map{File: "old-monolith.ts"}
	m.Add(sourcemap.Mapping{GenLine: 4, Source: source, SrcLine: 6, SrcCol: 1})
	names, err := writeSourceArtifacts(out, []artifact.File{
		{Path: "packages/value.go", Source: []byte("package generated\n"), Lines: map[int]token.Position{5: {Filename: source, Line: 7, Column: 2}}},
		{Path: "packages/value.ts", Source: []byte("export {};\n"), SourceMap: m},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(names, []string{"packages/value.go", "packages/value.go.lines", "packages/value.ts", "packages/value.ts.map"}) {
		t.Fatalf("missing map artifact: %v", names)
	}
	lines, err := os.ReadFile(filepath.Join(out, "packages/value.go.lines"))
	if err != nil || string(lines) != "5\t../../source.go:7:2\n" {
		t.Fatalf("map not relative to native file: %s, %v", lines, err)
	}
	data, err := os.ReadFile(filepath.Join(out, "packages/value.ts.map"))
	if err != nil {
		t.Fatal(err)
	}
	var actual struct {
		File    string
		Sources []string
	}
	if err := json.Unmarshal(data, &actual); err != nil {
		t.Fatal(err)
	}
	if actual.File != "value.ts" || !reflect.DeepEqual(actual.Sources, []string{"../../source.go"}) {
		t.Fatalf("wrong native source map: %+v", actual)
	}
	if m.File != "old-monolith.ts" {
		t.Fatal("writing mutated emitter metadata")
	}
}
