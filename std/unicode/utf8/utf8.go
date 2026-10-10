// Package utf8 encodes and decodes UTF-8 text.
package utf8

const (
	RuneError = '�'
	RuneSelf  = 0x80
	MaxRune   = '\U0010FFFF'
	UTFMax    = 4
)

const (
	surrogateMin = 0xD800
	surrogateMax = 0xDFFF
)

func decode(p []byte, s string, isString bool) (rune, int) {
	n := len(p)
	if isString {
		n = len(s)
	}
	if n < 1 {
		return RuneError, 0
	}
	at := func(i int) byte {
		if isString {
			return s[i]
		}
		return p[i]
	}
	c0 := at(0)
	if c0 < RuneSelf {
		return rune(c0), 1
	}
	size := 0
	var r, lo rune
	switch {
	case c0&0xE0 == 0xC0:
		size, r, lo = 2, rune(c0&0x1F), 0x80
	case c0&0xF0 == 0xE0:
		size, r, lo = 3, rune(c0&0x0F), 0x800
	case c0&0xF8 == 0xF0:
		size, r, lo = 4, rune(c0&0x07), 0x10000
	default:
		return RuneError, 1
	}
	if n < size {
		return RuneError, 1
	}
	for i := 1; i < size; i++ {
		c := at(i)
		if c&0xC0 != 0x80 {
			return RuneError, 1
		}
		r = r<<6 | rune(c&0x3F)
	}
	if r < lo || r > MaxRune || surrogateMin <= r && r <= surrogateMax {
		return RuneError, 1
	}
	return r, size
}

// DecodeRune unpacks the first UTF-8 encoding in p and returns the rune and
// its width in bytes. It returns (RuneError, 0) for empty input and
// (RuneError, 1) for an invalid encoding.
func DecodeRune(p []byte) (rune, int) { return decode(p, "", false) }

// DecodeRuneInString is like DecodeRune but its input is a string.
func DecodeRuneInString(s string) (rune, int) { return decode(nil, s, true) }

// DecodeLastRune unpacks the last UTF-8 encoding in p.
func DecodeLastRune(p []byte) (rune, int) {
	end := len(p)
	if end == 0 {
		return RuneError, 0
	}
	start := end - 1
	if p[start] < RuneSelf {
		return rune(p[start]), 1
	}
	lim := end - UTFMax
	if lim < 0 {
		lim = 0
	}
	for start--; start >= lim; start-- {
		if RuneStart(p[start]) {
			break
		}
	}
	if start < 0 {
		start = 0
	}
	r, size := DecodeRune(p[start:end])
	if start+size != end {
		return RuneError, 1
	}
	return r, size
}

// DecodeLastRuneInString is like DecodeLastRune but its input is a string.
func DecodeLastRuneInString(s string) (rune, int) {
	end := len(s)
	if end == 0 {
		return RuneError, 0
	}
	start := end - 1
	if s[start] < RuneSelf {
		return rune(s[start]), 1
	}
	lim := end - UTFMax
	if lim < 0 {
		lim = 0
	}
	for start--; start >= lim; start-- {
		if RuneStart(s[start]) {
			break
		}
	}
	if start < 0 {
		start = 0
	}
	r, size := DecodeRuneInString(s[start:end])
	if start+size != end {
		return RuneError, 1
	}
	return r, size
}

// FullRune reports whether the bytes in p begin with a full UTF-8 encoding
// of a rune. An invalid encoding is considered a full rune since it will
// convert as a width-1 error rune.
func FullRune(p []byte) bool {
	if len(p) == 0 {
		return false
	}
	c := p[0]
	need := 0
	switch {
	case c < 0xC2:
		return true
	case c < 0xE0:
		need = 2
	case c < 0xF0:
		need = 3
	case c < 0xF5:
		need = 4
	default:
		return true
	}
	if len(p) >= need {
		return true
	}
	lo, hi := byte(0x80), byte(0xBF)
	switch c {
	case 0xE0:
		lo = 0xA0
	case 0xED:
		hi = 0x9F
	case 0xF0:
		lo = 0x90
	case 0xF4:
		hi = 0x8F
	}
	if len(p) > 1 && (p[1] < lo || hi < p[1]) {
		return true
	}
	return len(p) > 2 && (p[2] < 0x80 || 0xBF < p[2])
}

// FullRuneInString is like FullRune but its input is a string.
func FullRuneInString(s string) bool { return FullRune([]byte(s)) }

// RuneStart reports whether b could be the first byte of an encoded rune.
func RuneStart(b byte) bool { return b&0xC0 != 0x80 }

// RuneLen returns the number of bytes needed to encode r, or -1 if r is not
// a valid Unicode scalar value.
func RuneLen(r rune) int {
	switch {
	case r < 0:
		return -1
	case r < 0x80:
		return 1
	case r < 0x800:
		return 2
	case surrogateMin <= r && r <= surrogateMax:
		return -1
	case r < 0x10000:
		return 3
	case r <= MaxRune:
		return 4
	}
	return -1
}

// ValidRune reports whether r can be legally encoded as UTF-8.
func ValidRune(r rune) bool {
	return 0 <= r && r < surrogateMin || surrogateMax < r && r <= MaxRune
}

// AppendRune appends the UTF-8 encoding of r to p. Invalid runes encode as
// RuneError.
func AppendRune(p []byte, r rune) []byte {
	if !ValidRune(r) {
		r = RuneError
	}
	switch {
	case r < 0x80:
		return append(p, byte(r))
	case r < 0x800:
		return append(p, byte(0xC0|r>>6), byte(0x80|r&0x3F))
	case r < 0x10000:
		return append(p, byte(0xE0|r>>12), byte(0x80|r>>6&0x3F), byte(0x80|r&0x3F))
	}
	return append(p, byte(0xF0|r>>18), byte(0x80|r>>12&0x3F), byte(0x80|r>>6&0x3F), byte(0x80|r&0x3F))
}

// EncodeRune writes the UTF-8 encoding of r into p, which must be large
// enough, and returns the number of bytes written.
func EncodeRune(p []byte, r rune) int {
	b := AppendRune(nil, r)
	for i := len(b) - 1; i >= 0; i-- {
		p[i] = b[i]
	}
	return len(b)
}

// RuneCount returns the number of runes in p, counting each invalid byte as
// one rune.
func RuneCount(p []byte) int {
	n := 0
	for i := 0; i < len(p); n++ {
		_, size := DecodeRune(p[i:])
		i += size
	}
	return n
}

// RuneCountInString is like RuneCount but its input is a string.
func RuneCountInString(s string) int {
	n := 0
	for i := 0; i < len(s); n++ {
		_, size := DecodeRuneInString(s[i:])
		i += size
	}
	return n
}

// Valid reports whether p consists entirely of valid UTF-8 encodings.
func Valid(p []byte) bool {
	for i := 0; i < len(p); {
		r, size := DecodeRune(p[i:])
		if r == RuneError && size == 1 {
			return false
		}
		i += size
	}
	return true
}

// ValidString reports whether s consists entirely of valid UTF-8 encodings.
func ValidString(s string) bool {
	for i := 0; i < len(s); {
		r, size := DecodeRuneInString(s[i:])
		if r == RuneError && size == 1 {
			return false
		}
		i += size
	}
	return true
}
