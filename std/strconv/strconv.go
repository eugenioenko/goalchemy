// Package strconv converts between integers, Booleans, and their string forms.
package strconv

import "github.com/eugenioenko/goalchemy/lib/errors"

// IntSize is the size in bits of an int or uint value.
const IntSize = 64

const digits = "0123456789abcdefghijklmnopqrstuvwxyz"

// ErrRange indicates that a value is out of range for the target type.
var ErrRange = errors.New("value out of range")

// ErrSyntax indicates that a value does not have the right syntax for the target type.
var ErrSyntax = errors.New("invalid syntax")

// NumError records a failed conversion.
type NumError struct {
	Func string
	Num  string
	Err  error
}

func (e *NumError) Error() string {
	return "strconv." + e.Func + ": parsing " + Quote(e.Num) + ": " + e.Err.Error()
}

func (e *NumError) Unwrap() error { return e.Err }

func syntaxError(fn, s string) *NumError { return &NumError{Func: fn, Num: s, Err: ErrSyntax} }

func rangeError(fn, s string) *NumError { return &NumError{Func: fn, Num: s, Err: ErrRange} }

func baseError(fn, s string, base int) *NumError {
	return &NumError{Func: fn, Num: s, Err: errors.New("invalid base " + Itoa(base))}
}

func bitSizeError(fn, s string, bitSize int) *NumError {
	return &NumError{Func: fn, Num: s, Err: errors.New("invalid bit size " + Itoa(bitSize))}
}

// Itoa returns the decimal form of i.
func Itoa(i int) string { return FormatInt(int64(i), 10) }

// Atoi parses a decimal int, accepting an optional sign.
func Atoi(s string) (int, error) {
	n, err := ParseInt(s, 10, 0)
	if err != nil {
		if ne, ok := err.(*NumError); ok {
			ne.Func = "Atoi"
		}
	}
	return int(n), err
}

// FormatInt returns i in the given base, 2 to 36, using lowercase letters.
func FormatInt(i int64, base int) string {
	if i >= 0 {
		return FormatUint(uint64(i), base)
	}
	return "-" + FormatUint(uint64(-(i+1))+1, base)
}

// FormatUint returns u in the given base, 2 to 36, using lowercase letters.
func FormatUint(u uint64, base int) string {
	if base < 2 || base > 36 {
		panic("strconv: illegal AppendInt/FormatInt base")
	}
	var buf [64]byte
	n := len(buf)
	b := uint64(base)
	for u >= b {
		n--
		buf[n] = digits[u%b]
		u /= b
	}
	n--
	buf[n] = digits[u]
	return string(buf[n:])
}

// AppendInt appends the string form of i in the given base to dst.
func AppendInt(dst []byte, i int64, base int) []byte {
	return append(dst, FormatInt(i, base)...)
}

// AppendUint appends the string form of u in the given base to dst.
func AppendUint(dst []byte, u uint64, base int) []byte {
	return append(dst, FormatUint(u, base)...)
}

// FormatBool returns "true" or "false".
func FormatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// ParseBool accepts 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, and False.
func ParseBool(str string) (bool, error) {
	switch str {
	case "1", "t", "T", "true", "TRUE", "True":
		return true, nil
	case "0", "f", "F", "false", "FALSE", "False":
		return false, nil
	}
	return false, syntaxError("ParseBool", str)
}

func lower(c byte) byte { return c | ('x' - 'X') }

func underscoreOK(s string) bool {
	saw := '^'
	i := 0
	if len(s) >= 1 && (s[0] == '-' || s[0] == '+') {
		s = s[1:]
	}
	hex := false
	if len(s) >= 2 && s[0] == '0' && (lower(s[1]) == 'b' || lower(s[1]) == 'o' || lower(s[1]) == 'x') {
		i = 2
		saw = '0'
		hex = lower(s[1]) == 'x'
	}
	for ; i < len(s); i++ {
		if '0' <= s[i] && s[i] <= '9' || hex && 'a' <= lower(s[i]) && lower(s[i]) <= 'f' {
			saw = '0'
			continue
		}
		if s[i] == '_' {
			if saw != '0' {
				return false
			}
			saw = '_'
			continue
		}
		if saw == '_' {
			return false
		}
		saw = '!'
	}
	return saw != '_'
}

// ParseUint parses s in the given base (0 or 2 to 36) and checks that the
// result fits in bitSize bits (0 means 64). Base 0 accepts 0b, 0o, 0, and 0x
// prefixes and underscores between digits.
func ParseUint(s string, base int, bitSize int) (uint64, error) {
	const fn = "ParseUint"
	if s == "" {
		return 0, syntaxError(fn, s)
	}
	base0 := base == 0
	s0 := s
	switch {
	case 2 <= base && base <= 36:
	case base == 0:
		base = 10
		if s[0] == '0' {
			switch {
			case len(s) >= 3 && lower(s[1]) == 'b':
				base = 2
				s = s[2:]
			case len(s) >= 3 && lower(s[1]) == 'o':
				base = 8
				s = s[2:]
			case len(s) >= 3 && lower(s[1]) == 'x':
				base = 16
				s = s[2:]
			default:
				base = 8
				s = s[1:]
			}
		}
	default:
		return 0, baseError(fn, s0, base)
	}
	if bitSize == 0 {
		bitSize = IntSize
	} else if bitSize < 0 || bitSize > 64 {
		return 0, bitSizeError(fn, s0, bitSize)
	}
	var cutoff uint64
	switch base {
	case 10:
		cutoff = 1844674407370955162
	case 16:
		cutoff = 1 << 60
	default:
		cutoff = 18446744073709551615/uint64(base) + 1
	}
	maxVal := uint64(1)<<uint(bitSize) - 1
	if bitSize == 64 {
		maxVal = 18446744073709551615
	}
	underscores := false
	var n uint64
	for i := 0; i < len(s); i++ {
		c := s[i]
		var d byte
		switch {
		case c == '_' && base0:
			underscores = true
			continue
		case '0' <= c && c <= '9':
			d = c - '0'
		case 'a' <= lower(c) && lower(c) <= 'z':
			d = lower(c) - 'a' + 10
		default:
			return 0, syntaxError(fn, s0)
		}
		if d >= byte(base) {
			return 0, syntaxError(fn, s0)
		}
		if n >= cutoff {
			return maxVal, rangeError(fn, s0)
		}
		n *= uint64(base)
		n1 := n + uint64(d)
		if n1 < n || n1 > maxVal {
			return maxVal, rangeError(fn, s0)
		}
		n = n1
	}
	if underscores && !underscoreOK(s0) {
		return 0, syntaxError(fn, s0)
	}
	return n, nil
}

// ParseInt parses a signed integer like ParseUint and checks that it fits in
// bitSize bits (0 means 64).
func ParseInt(s string, base int, bitSize int) (int64, error) {
	const fn = "ParseInt"
	if s == "" {
		return 0, syntaxError(fn, s)
	}
	s0 := s
	neg := false
	if s[0] == '+' {
		s = s[1:]
	} else if s[0] == '-' {
		neg = true
		s = s[1:]
	}
	un, err := ParseUint(s, base, bitSize)
	if err != nil {
		ne, ok := err.(*NumError)
		if ok && ne.Err != ErrRange {
			ne.Func = fn
			ne.Num = s0
			return 0, ne
		}
	}
	if bitSize == 0 {
		bitSize = IntSize
	}
	cutoff := uint64(1) << uint(bitSize-1)
	if !neg && un >= cutoff {
		return int64(cutoff - 1), rangeError(fn, s0)
	}
	if neg && un > cutoff {
		return -int64(cutoff - 1) - 1, rangeError(fn, s0)
	}
	n := int64(un)
	if neg {
		n = -n
	}
	return n, nil
}

// Quote returns s as a double-quoted Go string literal, escaping
// non-printable runes and invalid UTF-8 bytes.
func Quote(s string) string { return quote(s, false) }

// QuoteToASCII is like Quote but escapes every non-ASCII rune.
func QuoteToASCII(s string) string { return quote(s, true) }

func quote(s string, ascii bool) string {
	out := make([]byte, 0, len(s)+2)
	out = append(out, '"')
	for i := 0; i < len(s); {
		r, size := decodeRune(s[i:])
		if r == 0xFFFD && size == 1 {
			out = append(out, '\\', 'x', digits[s[i]>>4], digits[s[i]&15])
			i++
			continue
		}
		out = appendEscapedRune(out, r, ascii)
		i += size
	}
	return string(append(out, '"'))
}

func appendEscapedRune(out []byte, r rune, ascii bool) []byte {
	if r == '"' || r == '\\' {
		return append(out, '\\', byte(r))
	}
	if ascii && r < 0x80 && isPrint(r) || !ascii && isPrint(r) {
		return appendRune(out, r)
	}
	switch r {
	case '\a':
		return append(out, '\\', 'a')
	case '\b':
		return append(out, '\\', 'b')
	case '\f':
		return append(out, '\\', 'f')
	case '\n':
		return append(out, '\\', 'n')
	case '\r':
		return append(out, '\\', 'r')
	case '\t':
		return append(out, '\\', 't')
	case '\v':
		return append(out, '\\', 'v')
	}
	switch {
	case r < ' ' || r == 0x7f:
		return append(out, '\\', 'x', digits[byte(r)>>4], digits[byte(r)&15])
	case r < 0x10000:
		out = append(out, '\\', 'u')
		for s := 12; s >= 0; s -= 4 {
			out = append(out, digits[r>>uint(s)&15])
		}
		return out
	}
	out = append(out, '\\', 'U')
	for s := 28; s >= 0; s -= 4 {
		out = append(out, digits[r>>uint(s)&15])
	}
	return out
}

func isPrint(r rune) bool {
	lo, hi := 0, len(printLo)
	for lo < hi {
		m := lo + (hi-lo)/2
		if printHi[m] < r {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return lo < len(printLo) && printLo[lo] <= r
}

func decodeRune(s string) (rune, int) {
	c0 := s[0]
	if c0 < 0x80 {
		return rune(c0), 1
	}
	n := 0
	var r rune
	var lo rune
	switch {
	case c0&0xe0 == 0xc0:
		n, r, lo = 2, rune(c0&0x1f), 0x80
	case c0&0xf0 == 0xe0:
		n, r, lo = 3, rune(c0&0x0f), 0x800
	case c0&0xf8 == 0xf0:
		n, r, lo = 4, rune(c0&0x07), 0x10000
	default:
		return 0xFFFD, 1
	}
	if len(s) < n {
		return 0xFFFD, 1
	}
	for i := 1; i < n; i++ {
		if s[i]&0xc0 != 0x80 {
			return 0xFFFD, 1
		}
		r = r<<6 | rune(s[i]&0x3f)
	}
	if r < lo || r > 0x10ffff || 0xd800 <= r && r <= 0xdfff {
		return 0xFFFD, 1
	}
	return r, n
}

func appendRune(out []byte, r rune) []byte {
	switch {
	case r < 0x80:
		return append(out, byte(r))
	case r < 0x800:
		return append(out, byte(0xc0|r>>6), byte(0x80|r&0x3f))
	case r < 0x10000:
		return append(out, byte(0xe0|r>>12), byte(0x80|r>>6&0x3f), byte(0x80|r&0x3f))
	}
	return append(out, byte(0xf0|r>>18), byte(0x80|r>>12&0x3f), byte(0x80|r>>6&0x3f), byte(0x80|r&0x3f))
}
