package rt

// StringToBytes converts with capacity equal to length.
func StringToBytes(s string) []byte {
	b := make([]byte, len(s))
	copy(b, s)
	return b
}
