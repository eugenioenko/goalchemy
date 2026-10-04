package ir

import (
	"go/constant"
	"go/token"
	"go/types"
	"testing"
)

func TestFloatWidthsAndExactConstantRounding(t *testing.T) {
	ts := NewTypes()
	f32, f64 := ts.Basic(types.Float32), ts.Basic(types.Float64)
	if f32.Kind != KFloat || f32.FloatBits != 32 || f64.FloatBits != 64 || TypeString(f32) != "float32" {
		t.Fatal("float type width/name lost")
	}
	// Just above the midpoint: converting to binary64 first erases the tail.
	v := constant.MakeFromLiteral("0x1.00000100000000000001p0", token.FLOAT, 0)
	if got := FloatLiteral(&Const{Type: f32, Val: v}); got != "1.0000001192092896" {
		t.Fatalf("direct binary32 rounding: %s", got)
	}
	if got := FloatLiteral(&Const{Type: f64, Val: v}); got != "1.0000000596046448" {
		t.Fatalf("binary64 rounding: %s", got)
	}
	if got := FloatLiteral(&Const{Type: f32, Val: constant.MakeInt64(0)}); got != "0.0" {
		t.Fatal(got)
	}
}

func TestFloatIntegerValuedConstantVerifier(t *testing.T) {
	ts := NewTypes()
	f := &Func{Name: "floatconstant"}
	b := f.NewBlock("")
	l := f.NewLocal("x", ts.Basic(types.Float32), LVar)
	b.Instrs = []Instr{&Assign{Dst: l, Src: &Const{Type: l.Type, Val: constant.MakeInt64(1)}}}
	b.Term = &Return{}
	if err := Verify(f); err != nil {
		t.Fatal(err)
	}
	l.Type = ts.Bool()
	b.Instrs[0].(*Assign).Src.(*Const).Type = l.Type
	if err := Verify(f); err == nil {
		t.Fatal("integer constant admitted as bool")
	}
}
