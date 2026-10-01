package rt

// GrowCap returns Goalchemy's selected capacity for growing old to required.
func GrowCap(old, required int) int {
	const maxCap = int(^uint(0) >> 1)
	c := maxCap
	if old <= maxCap/2 {
		c = 2 * old
	}
	return max(required, max(1, c))
}

// Append implements append(s, vs...) with Goalchemy's growth rule.
func Append[T any](s []T, vs ...T) []T {
	n := len(s) + len(vs)
	if n < len(s) {
		panic(RuntimeError("growslice: len out of range"))
	}
	if n <= cap(s) {
		r := s[:n]
		copy(r[len(s):], vs)
		return r
	}
	if len(vs) == 0 {
		return s
	}
	r := make([]T, n, GrowCap(cap(s), n))
	copy(r, s)
	copy(r[len(s):], vs)
	return r
}

// AppendString implements append(b, s...) for byte slices.
func AppendString[B ~[]byte](b B, s string) B {
	return B(Append([]byte(b), []byte(s)...))
}
