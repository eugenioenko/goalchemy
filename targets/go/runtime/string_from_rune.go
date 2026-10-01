package rt

func StringFromRune[I Integer](r I) string {
	if int64(r) < 0 || uint64(r) > 0x10FFFF {
		return "�"
	}
	return string(rune(r))
}
