package rt

import (
	"math"
	"reflect"
	"testing"
)

func TestFloatPanicFormatting(t *testing.T) {
	type F32 float32
	type F64 float64
	t32, t64 := reflect.TypeOf(F32(0)), reflect.TypeOf(F64(0))
	old32, old64 := names[t32], names[t64]
	RegisterType(t32, "example.F32")
	RegisterType(t64, "example.F64")
	defer func() {
		if old32 == "" {
			delete(names, t32)
		} else {
			names[t32] = old32
		}
		if old64 == "" {
			delete(names, t64)
		} else {
			names[t64] = old64
		}
	}()
	negative := math.Copysign(0, -1)
	for _, c := range []struct {
		value any
		want  string
	}{
		{float32(1.23456789), "1.2345679"}, {float64(1.23456789), "1.23456789"},
		{F32(1.23456789), "example.F32(1.2345679)"}, {F64(1.23456789), "example.F64(1.23456789)"},
		{float32(negative), "-0"}, {negative, "-0"}, {F32(negative), "example.F32(-0)"}, {F64(negative), "example.F64(-0)"},
		{float32(math.Inf(1)), "+Inf"}, {math.Inf(-1), "-Inf"}, {F32(math.Inf(-1)), "example.F32(-Inf)"}, {F64(math.Inf(1)), "example.F64(+Inf)"},
		{float32(math.NaN()), "NaN"}, {math.NaN(), "NaN"}, {F32(math.NaN()), "example.F32(NaN)"}, {F64(math.NaN()), "example.F64(NaN)"},
	} {
		if got := FormatPanic(c.value); got != c.want {
			t.Errorf("%T: got %q want %q", c.value, got, c.want)
		}
	}
}

type floatPayloadError struct {
	Small  float32
	Wide   float64
	Values []float32
	Nested [][]float64
	Pair   [2]float32
}

func (*floatPayloadError) Error() string { return "float failure" }

func TestFloatErrorPayloadSnapshot(t *testing.T) {
	negative := math.Copysign(0, -1)
	source := &floatPayloadError{Small: float32(negative), Wide: math.NaN(), Values: []float32{1.25, float32(math.Inf(1))}, Nested: [][]float64{{2.5, negative}}, Pair: [2]float32{3.75, 4.25}}
	got, ok := SnapshotError(source).(*floatPayloadError)
	if !ok {
		t.Fatalf("typed float error rejected: %T", SnapshotError(source))
	}
	if got == source || !math.Signbit(float64(got.Small)) || !math.IsNaN(got.Wide) || !math.IsInf(float64(got.Values[1]), 1) || !math.Signbit(got.Nested[0][1]) || got.Pair != source.Pair {
		t.Fatal("float error scalar payload changed")
	}
	got.Values[0] = 9
	got.Nested[0][0] = 10
	got.Pair[0] = 11
	if source.Values[0] != 1.25 || source.Nested[0][0] != 2.5 || source.Pair[0] != 3.75 {
		t.Fatal("float error snapshot aliases source")
	}
	again := SnapshotError(source).(*floatPayloadError)
	if again.Values[0] != 1.25 || again.Nested[0][0] != 2.5 {
		t.Fatal("later error snapshot shares output")
	}
	empty := SnapshotError(&floatPayloadError{Values: []float32{}, Nested: nil}).(*floatPayloadError)
	if empty.Values == nil || empty.Nested != nil {
		t.Fatal("float nil/empty error shape lost")
	}
}
