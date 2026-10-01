package language

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/project"
)

// TestFeatureManifest checks that every supported or rejected feature names
// fixtures that exist, and that supported targets are tested here.
func TestFeatureManifest(t *testing.T) {
	f, err := project.LoadFeatures(os.DirFS("../.."))
	if err != nil {
		t.Fatal(err)
	}
	tested := map[string]bool{}
	for _, tg := range targets {
		tested[tg] = true
	}
	for _, ft := range f.Features {
		for _, fx := range ft.Fixtures {
			if _, err := os.Stat(filepath.Join("testdata", fx, "main.go")); err != nil {
				t.Errorf("feature %s: fixture %s does not exist", ft.ID, fx)
			}
		}
		if ft.Status == "supported" {
			for _, tg := range ft.Targets {
				if !tested[tg] {
					t.Errorf("feature %s claims target %s, which the language suite does not run", ft.ID, tg)
				}
			}
		}
	}
}
