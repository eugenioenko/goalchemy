// Package corpus replays saved regressions: each directory holds the
// reducing source, its metadata, and optionally a golden observation.
package corpus

import (
	"os"
	"path/filepath"
	"testing"

	"goalchemy/internal/project"
	"goalchemy/internal/testutil"
)

var targets = []string{"go", "typescript", "python", "java", "csharp", "rust", "c"}

func TestRegressions(t *testing.T) {
	targets := targets
	if testing.Short() {
		targets = targets[:2]
	}
	fixtures, err := testutil.Discover(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fixtures {
		t.Run(f.Name, func(t *testing.T) {
			t.Parallel()
			if _, err := os.Stat(filepath.Join(f.Dir, "meta.yaml")); err != nil {
				t.Fatal("regression is missing meta.yaml")
			}
			if _, err := project.LoadMeta(filepath.Join(f.Dir, "meta.yaml")); err != nil {
				t.Fatal(err)
			}
			testutil.RunFixture(t, f, targets)
		})
	}
}
