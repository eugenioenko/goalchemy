package rt

// StringToRunes converts with capacity equal to length.
func StringToRunes(s string) []rune {
	n := 0
	for range s {
		n++
	}
	r := make([]rune, 0, n)
	for _, c := range s {
		r = append(r, c)
	}
	return r
}
