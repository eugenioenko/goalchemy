package rt

type floatNumeric interface{ Integer | Floating }

// Width-specific conversions use the source numeric types directly, preserving
// integer-to-float32 rounding without an intermediate float64 conversion.
func FloatConvert[T floatNumeric, U floatNumeric](x T) U { return U(x) }
