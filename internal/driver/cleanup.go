package driver

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/eugenioenko/goalchemy/internal/link"
)

// managedFiles validates the previous/current inventory before any deletion.
// Only explicit generated/runtime members are managed; caller files are not.
func managedFiles(out string) (map[string]bool, error) {
	data, err := os.ReadFile(filepath.Join(out, "goalchemy.manifest.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var manifest link.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	files := map[string]bool{}
	for _, name := range append(manifest.GeneratedFiles, manifest.RuntimeFiles...) {
		if name == "." || !fs.ValidPath(name) || strings.Contains(name, "\\") || strings.Contains(name, ":") || name == "goalchemy.manifest.json" {
			return nil, fmt.Errorf("invalid managed output path %q", name)
		}
		// A symlink below the output root could make cleanup reach caller
		// storage outside this tree. Validate each existing path component.
		current := out
		parts := strings.Split(name, "/")
		for i, component := range parts {
			current = filepath.Join(current, component)
			info, err := os.Lstat(current)
			if os.IsNotExist(err) {
				break
			}
			if err != nil {
				return nil, err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("managed output path traverses symlink %q", name)
			}
			if i == len(parts)-1 && info.IsDir() {
				return nil, fmt.Errorf("managed output member is a directory %q", name)
			}
		}
		files[name] = true
	}
	return files, nil
}

func cleanObsoleteFiles(out string, previous map[string]bool) error {
	if previous == nil {
		return nil
	}
	current, err := managedFiles(out)
	if err != nil {
		return err
	}
	if current == nil {
		return fmt.Errorf("successful emission did not write an output manifest")
	}
	var obsolete []string
	for name := range previous {
		if !current[name] {
			obsolete = append(obsolete, name)
		}
	}
	sort.Strings(obsolete)
	for _, name := range obsolete {
		if err := os.Remove(filepath.Join(out, filepath.FromSlash(name))); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
