package rt

import "unicode/utf8"

// DecodeRune decodes the rune at byte offset i as range over string does.
func DecodeRune(s string, i int) (int32, int) {
	r, w := utf8.DecodeRuneInString(s[i:])
	return r, w
}
