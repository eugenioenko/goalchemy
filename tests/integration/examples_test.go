package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/project"
	"github.com/eugenioenko/goalchemy/internal/testutil"
)

// TestExamples builds each example from its goalchemy.yaml and compares
// every target with native Go.
func TestExamples(t *testing.T) {
	dirs, _ := filepath.Glob("../../examples/*")
	if len(dirs) == 0 {
		t.Fatal("no examples")
	}
	for _, dir := range dirs {
		cfg, err := project.Load(filepath.Join(dir, project.FileName))
		if err != nil {
			t.Fatal(err)
		}
		f := testutil.Fixture{Name: filepath.Base(dir), Dir: dir}
		src, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		f.Unordered = testutil.HasDirective(string(src), "unordered")
		t.Run(f.Name, func(t *testing.T) {
			t.Parallel()
			targets := cfg.TargetNames()
			if testing.Short() {
				targets = []string{"go", "typescript"}
			}
			testutil.RunFixture(t, f, targets)
		})
	}
}
