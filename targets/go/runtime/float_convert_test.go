package rt

import (
	"math"
	"testing"
)

func checkFloatIntegerWidth[T Integer](t *testing.T) {
	t.Helper()
	if got := FloatConvert[T, float32](T(42)); got != 42 {
		t.Errorf("%T to float32: got %v want 42", T(0), got)
	}
	if got := FloatConvert[T, float64](T(42)); got != 42 {
		t.Errorf("%T to float64: got %v want 42", T(0), got)
	}
	if got := FloatConvert[float32, T](42.75); got != 42 {
		t.Errorf("float32 to %T: got %v want 42", T(0), got)
	}
	if got := FloatConvert[float64, T](42.75); got != 42 {
		t.Errorf("float64 to %T: got %v want 42", T(0), got)
	}
}

func TestFloatConvertIntegerWidths(t *testing.T) {
	checkFloatIntegerWidth[int8](t)
	checkFloatIntegerWidth[int16](t)
	checkFloatIntegerWidth[int32](t)
	checkFloatIntegerWidth[int64](t)
	checkFloatIntegerWidth[int](t)
	checkFloatIntegerWidth[uint8](t)
	checkFloatIntegerWidth[uint16](t)
	checkFloatIntegerWidth[uint32](t)
	checkFloatIntegerWidth[uint64](t)
	checkFloatIntegerWidth[uint](t)
}

func TestFloatConvertDirectRounding(t *testing.T) {
	// These values lie just above a float32 midpoint but round to that midpoint
	// in float64. An intermediate float64 would round to the wrong float32.
	if got := math.Float32bits(FloatConvert[int64, float32](4611686293305294849)); got != 0x5e800001 {
		t.Errorf("signed midpoint: got %08x want 5e800001", got)
	}
	if got := math.Float32bits(FloatConvert[int64, float32](-4611686293305294849)); got != 0xde800001 {
		t.Errorf("negative midpoint: got %08x want de800001", got)
	}
	if got := math.Float32bits(FloatConvert[uint64, float32](9223372586610589697)); got != 0x5f000001 {
		t.Errorf("unsigned midpoint: got %08x want 5f000001", got)
	}
	type Signed int64
	type Single float32
	if got := math.Float32bits(float32(FloatConvert[Signed, Single](4611686293305294849))); got != 0x5e800001 {
		t.Errorf("named numeric types: got %08x want 5e800001", got)
	}
	if got := FloatConvert[float64, int32](-42.75); got != -42 {
		t.Errorf("truncation: got %d want -42", got)
	}
	if got := FloatConvert[float64, int8](255.9); got != -1 {
		t.Errorf("narrow wrap: got %d want -1", got)
	}
	if got := FloatConvert[float64, uint64](-1); got != math.MaxUint64 {
		t.Errorf("unsigned wrap: got %d want MaxUint64", got)
	}
}

func TestFloatConvertExceptionalAMD64Profile(t *testing.T) {
	// These implementation-dependent results are the contract's selected
	// Go linux/amd64 profile, rather than portable Go language guarantees.
	for _, x := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), math.MaxFloat64} {
		if got := FloatConvert[float64, int32](x); got != math.MinInt32 {
			t.Errorf("int32(%v): got %d want MinInt32", x, got)
		}
		if got := FloatConvert[float64, uint64](x); got != uint64(1)<<63 {
			t.Errorf("uint64(%v): got %d want 1<<63", x, got)
		}
	}
}
