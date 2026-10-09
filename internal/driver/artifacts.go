package driver

import (
	"path/filepath"

	"github.com/eugenioenko/goalchemy/internal/emit/artifact"
	"github.com/eugenioenko/goalchemy/internal/link"
)

// writeSourceArtifacts writes each native source and its own diagnostic map.
// The returned names include sidecars, so manifests describe the complete tree.
func writeSourceArtifacts(out string, files []artifact.File) ([]string, error) {
	var names []string
	for _, file := range files {
		if err := link.WriteFile(out, file.Path, file.Source); err != nil {
			return nil, err
		}
		names = append(names, file.Path)
		dir, err := filepath.Abs(filepath.Dir(filepath.Join(out, file.Path)))
		if err != nil {
			return nil, err
		}
		if len(file.Lines) > 0 {
			name := file.Path + ".lines"
			if err := link.WriteFile(out, name, lineTable(file.Lines, dir)); err != nil {
				return nil, err
			}
			names = append(names, name)
		}
		if file.SourceMap != nil {
			m := *file.SourceMap
			m.File = filepath.Base(file.Path)
			data, err := m.JSON(dir)
			if err != nil {
				return nil, err
			}
			name := file.Path + ".map"
			if err := link.WriteFile(out, name, data); err != nil {
				return nil, err
			}
			names = append(names, name)
		}
	}
	return names, nil
}
