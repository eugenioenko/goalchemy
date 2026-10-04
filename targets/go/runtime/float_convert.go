package rt

// Width-specific conversions are emitted as explicit native Go conversions.
func FloatConvert[T Floating, U Floating](x T) U { return U(x) }
