package contracts_test

import (
	"github.com/eugenioenko/goalchemy/internal/contracts"
	"testing"
)

func TestFloatTypeFamiliesAndCollections(t *testing.T) {
	for _, source := range []string{"float32", "float64", "[]float32", "[2]float64", "map[float64][]float32", "chan float64", "*float32"} {
		typ, err := contracts.ParseType(source)
		if err != nil || !typ.Concrete() || !contracts.InFamily(typ, "any") {
			t.Fatalf("float collection %s: %v", source, err)
		}
		scalar := source == "float32" || source == "float64"
		if contracts.InFamily(typ, "type.float") != scalar {
			t.Fatalf("wrong float family: %s", source)
		}
	}
	for _, source := range []string{"int", "complex64", "complex128"} {
		typ, err := contracts.ParseType(source)
		if err != nil || contracts.InFamily(typ, "type.float") {
			t.Fatalf("unexpected float: %s", source)
		}
	}
}
