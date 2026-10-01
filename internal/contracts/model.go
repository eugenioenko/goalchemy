// Package contracts loads, validates, and normalizes the canonical runtime
// contract catalog and the target mappings that implement it.
package contracts

import "encoding/json"

type TypeParam struct {
	Name       string `json:"name"`
	Family     string `json:"family"`
	Comparable bool   `json:"comparable,omitempty"`
}

type Param struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Signature struct {
	Source         string      `json:"source"`
	Symbol         string      `json:"symbol,omitempty"`
	TypeParameters []TypeParam `json:"type_parameters"`
	Inputs         []Param     `json:"inputs"`
	Outputs        []Param     `json:"outputs"`
	Variadic       bool        `json:"variadic,omitempty"`
}

type Dependency struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type Binding struct {
	Name  string          `json:"name"`
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

type After struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}

type PanicExpect struct {
	Category string `json:"category"`
	Message  string `json:"message"`
}

type Expect struct {
	Results []json.RawMessage `json:"results,omitempty"`
	Panic   *PanicExpect      `json:"panic,omitempty"`
	After   []After           `json:"after,omitempty"`
}

type Case struct {
	Name   string            `json:"name"`
	Types  map[string]string `json:"types"`
	Let    []Binding         `json:"let"`
	Call   []string          `json:"call"`
	Expect Expect            `json:"expect"`
}

type Function struct {
	ID              string            `json:"id"`
	SchemaVersion   int               `json:"schema_version"`
	Version         string            `json:"version"`
	Summary         string            `json:"summary"`
	Signature       Signature         `json:"signature"`
	DependsOn       []Dependency      `json:"depends_on"`
	Suspension      string            `json:"suspension"`
	Determinism     string            `json:"determinism"`
	Behavior        map[string]string `json:"behavior"`
	PanicCategories []string          `json:"panic_categories,omitempty"`
	Notes           string            `json:"notes,omitempty"`
	Cases           []Case            `json:"cases"`

	Path string `json:"-"`
	Hash string `json:"-"`
}

type Type struct {
	ID            string            `json:"id"`
	SchemaVersion int               `json:"schema_version"`
	Version       string            `json:"version"`
	Family        string            `json:"family"`
	Summary       string            `json:"summary"`
	Semantics     map[string]string `json:"semantics"`
	Members       []string          `json:"members,omitempty"`
	Notes         string            `json:"notes,omitempty"`
	ExercisedBy   []string          `json:"exercised_by"`

	Path string `json:"-"`
	Hash string `json:"-"`
}

type Representation struct {
	Type            string `json:"type"`
	ContractVersion string `json:"contract_version"`
	Representation  string `json:"representation"`
}

type Implementation struct {
	ID              string   `json:"id"`
	ContractVersion string   `json:"contract_version"`
	Symbol          string   `json:"symbol"`
	Implementation  string   `json:"implementation"`
	Requires        []string `json:"requires,omitempty"`
}

type Environment struct {
	Runtime string `json:"runtime"`
	Minimum string `json:"minimum"`
	Build   string `json:"build,omitempty"`
}

type Target struct {
	SchemaVersion   int              `json:"schema_version"`
	Target          string           `json:"target"`
	Version         string           `json:"version"`
	Status          string           `json:"status"`
	Environment     Environment      `json:"environment"`
	Representations []Representation `json:"representations"`
	Functions       []Implementation `json:"functions"`
	SupportFiles    []string         `json:"support_files,omitempty"`

	Path string `json:"-"`
	// Dir is the target directory relative to the catalog root.
	Dir string `json:"-"`
}

func (t *Target) Function(id string) (Implementation, bool) {
	for _, f := range t.Functions {
		if f.ID == id {
			return f, true
		}
	}
	return Implementation{}, false
}
