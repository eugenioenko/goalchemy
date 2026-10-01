package contracts_test

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/eugenioenko/goalchemy"
	"github.com/eugenioenko/goalchemy/internal/catalog"
	"github.com/eugenioenko/goalchemy/internal/contracts"
)

func baseFS(t *testing.T) fstest.MapFS {
	m := fstest.MapFS{}
	err := fs.WalkDir(goalchemy.Assets, "specs", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(goalchemy.Assets, p)
		m[p] = &fstest.MapFile{Data: b}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestRepositoryCatalogIsValid(t *testing.T) {
	cat, ds := contracts.Load(goalchemy.Assets, contracts.LoadOptions{Resolve: catalog.Resolver(goalchemy.Assets)})
	for _, d := range ds {
		t.Error(d.String())
	}
	if len(cat.Functions) == 0 || len(cat.Types) == 0 {
		t.Fatal("empty catalog")
	}
}

const addHeader = `id: core.integer.add
schema_version: 1
version: 1.0.0
summary: test
signature:
  source: intrinsic
  type_parameters:
    - {name: T, family: type.integer}
  inputs:
    - {name: a, type: T}
    - {name: b, type: T}
  outputs:
    - {name: result, type: T}
suspension: never
determinism: deterministic
behavior: {zero: "", nil: "", bounds: "", conversions: "", overflow: "", mutations: "", aliasing: "", allocation: "", errors: "", panics: "", observations: ""}
`

func TestMalformedContracts(t *testing.T) {
	cases := map[string]struct {
		file, body, want string
	}{
		"unknown field": {"specs/runtime/core/integer_add.yaml", addHeader + "depends_on: [{id: type.integer, version: 1.0.0}]\nextra: 1\ncases: []\n", "GCC003"},
		"bad dep version": {"specs/runtime/core/integer_add.yaml", addHeader + "depends_on: [{id: type.integer, version: 9.0.0}]\n" +
			`cases: [{name: c, types: {T: int}, let: [{name: a, type: T, value: "1"}, {name: b, type: T, value: "2"}], call: [a, b], expect: {results: ["3"]}}]` + "\n", "dependency type.integer requires version 9.0.0"},
		"out of range value": {"specs/runtime/core/integer_add.yaml", addHeader + "depends_on: [{id: type.integer, version: 1.0.0}]\n" +
			`cases: [{name: c, types: {T: int8}, let: [{name: a, type: T, value: "128"}, {name: b, type: T, value: "2"}], call: [a, b], expect: {results: ["3"]}}]` + "\n", "outside the range of int8"},
		"non canonical int": {"specs/runtime/core/integer_add.yaml", addHeader + "depends_on: [{id: type.integer, version: 1.0.0}]\n" +
			`cases: [{name: c, types: {T: int}, let: [{name: a, type: T, value: "01"}, {name: b, type: T, value: "2"}], call: [a, b], expect: {results: ["3"]}}]` + "\n", "canonical decimal"},
		"wrong family": {"specs/runtime/core/integer_add.yaml", addHeader + "depends_on: [{id: type.integer, version: 1.0.0}]\n" +
			`cases: [{name: c, types: {T: string}, let: [], call: [], expect: {results: []}}]` + "\n", "not in family"},
		"undeclared panic": {"specs/runtime/core/integer_add.yaml", addHeader + "depends_on: [{id: type.integer, version: 1.0.0}]\n" +
			`cases: [{name: c, types: {T: int}, let: [{name: a, type: T, value: "1"}, {name: b, type: T, value: "2"}], call: [a, b], expect: {panic: {category: boom, message: x}}}]` + "\n", "not declared"},
		"duplicate key": {"specs/types/integer.yaml", "id: type.integer\nid: type.integer\n", "duplicate key"},
		"duplicate id":  {"specs/runtime/core/zz_dup.yaml", "", "duplicate function contract core.integer.sub"},
		"cycle":         {"specs/runtime/core/zz_cycle.yaml", "", "dependency cycle"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			m := baseFS(t)
			body := tc.body
			switch name {
			case "duplicate id":
				body = string(m["specs/runtime/core/integer_sub.yaml"].Data)
			case "cycle":
				sub := string(m["specs/runtime/core/integer_sub.yaml"].Data)
				sub = strings.Replace(sub, "depends_on:\n", "depends_on:\n  - {id: core.integer.neg, version: 1.0.0}\n", 1)
				m["specs/runtime/core/integer_sub.yaml"] = &fstest.MapFile{Data: []byte(sub)}
				neg := string(m["specs/runtime/core/integer_neg.yaml"].Data)
				neg = strings.Replace(neg, "depends_on:\n", "depends_on:\n  - {id: core.integer.sub, version: 1.0.0}\n", 1)
				m["specs/runtime/core/integer_neg.yaml"] = &fstest.MapFile{Data: []byte(neg)}
				body = ""
			}
			if body != "" {
				m[tc.file] = &fstest.MapFile{Data: []byte(body)}
			}
			_, ds := contracts.Load(m, contracts.LoadOptions{Resolve: catalog.Resolver(m)})
			var all []string
			for _, d := range ds {
				all = append(all, d.Code+" "+d.Message)
			}
			joined := strings.Join(all, "\n")
			if !strings.Contains(joined, tc.want) {
				t.Errorf("want %q in diagnostics:\n%s", tc.want, joined)
			}
		})
	}
}
