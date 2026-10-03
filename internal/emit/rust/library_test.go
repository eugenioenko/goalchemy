package rust

import (
	"github.com/eugenioenko/goalchemy/internal/ir"
	"testing"
)

func TestNativeBoundaryRejectsPointers(t *testing.T) {
	i := &ir.Type{Kind: ir.KInt, Int: ir.I64}
	p := &ir.Type{Kind: ir.KPointer, Elem: i}
	if libraryValue(p, map[*ir.Type]bool{}) {
		t.Fatal("borrowed source pointer accepted")
	}
	if !libraryValue(i, map[*ir.Type]bool{}) {
		t.Fatal("integer rejected")
	}
}
