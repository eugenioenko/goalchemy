package project

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"

	"github.com/eugenioenko/goalchemy/internal/contracts"
)

type GateStatus struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Targets []string `json:"targets"`
}

type Feature struct {
	ID       string   `json:"id"`
	Gate     string   `json:"gate"`
	Status   string   `json:"status"`
	Targets  []string `json:"targets"`
	Fixtures []string `json:"fixtures"`
}

type Features struct {
	SchemaVersion int          `json:"schema_version"`
	Release       string       `json:"release"`
	Gates         []GateStatus `json:"gates"`
	Features      []Feature    `json:"features"`
}

// LoadFeatures reads specs/features.yaml from the catalog filesystem.
func LoadFeatures(fsys fs.FS) (*Features, error) {
	raw, err := fs.ReadFile(fsys, "specs/features.yaml")
	if err != nil {
		return nil, err
	}
	v, err := contracts.ParseYAML(raw)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(v)
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var f Features
	if err := dec.Decode(&f); err != nil {
		return nil, err
	}
	for _, ft := range f.Features {
		switch ft.Status {
		case "supported", "planned", "rejected":
		default:
			return nil, fmt.Errorf("feature %s: unknown status %q", ft.ID, ft.Status)
		}
		if ft.Status == "supported" && (len(ft.Targets) == 0 || len(ft.Fixtures) == 0) {
			return nil, fmt.Errorf("feature %s: supported features need targets and fixtures", ft.ID)
		}
	}
	return &f, nil
}
