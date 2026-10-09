package fmt

import (
	"github.com/eugenioenko/goalchemy/std/strconv"
	"github.com/eugenioenko/goalchemy/std/unicode/utf8"
)

const (
	ldigits = "0123456789abcdefx"
	udigits = "0123456789ABCDEFX"
)

const (
	signed   = true
	unsigned = false
)

type fmtFlags struct {
	widPresent  bool
	precPresent bool
	minus       bool
	plus        bool
	sharp       bool
	space       bool
	zero        bool

	// %+v and %#v set plusV/sharpV and clear plus/sharp: they are distinct
	// flagless formats at the top level.
	plusV  bool
	sharpV bool
}

// formatter is the raw formatter used by Sprintf etc. It prints into a
// buffer that must be set up separately.
type formatter struct {
	buf *buffer

	fmtFlags

	wid  int
	prec int
}

func (f *formatter) clearflags() {
	f.fmtFlags = fmtFlags{}
	f.wid = 0
	f.prec = 0
}

func (f *formatter) init(buf *buffer) {
	f.buf = buf
	f.clearflags()
}

func (f *formatter) writePadding(n int) {
	if n <= 0 {
		return
	}
	padByte := byte(' ')
	if f.zero && !f.minus {
		padByte = byte('0')
	}
	for i := 0; i < n; i++ {
		f.buf.writeByte(padByte)
	}
}

func (f *formatter) pad(b []byte) {
	if !f.widPresent || f.wid == 0 {
		f.buf.write(b)
		return
	}
	width := f.wid - utf8.RuneCount(b)
	if !f.minus {
		f.writePadding(width)
		f.buf.write(b)
	} else {
		f.buf.write(b)
		f.writePadding(width)
	}
}

func (f *formatter) padString(s string) {
	if !f.widPresent || f.wid == 0 {
		f.buf.writeString(s)
		return
	}
	width := f.wid - utf8.RuneCountInString(s)
	if !f.minus {
		f.writePadding(width)
		f.buf.writeString(s)
	} else {
		f.buf.writeString(s)
		f.writePadding(width)
	}
}

func (f *formatter) fmtBoolean(v bool) {
	if v {
		f.padString("true")
	} else {
		f.padString("false")
	}
}

// fmtUnicode formats a uint64 as "U+0078" or with f.sharp set as "U+0078 'x'".
func (f *formatter) fmtUnicode(u uint64) {
	prec := 4
	if f.precPresent && f.prec > 4 {
		prec = f.prec
	}
	buf := make([]byte, 2+prec+2+utf8.UTFMax+1+16)
	i := len(buf)
	if f.sharp && u <= utf8.MaxRune && strconv.IsPrint(rune(u)) {
		i--
		buf[i] = '\''
		i -= utf8.RuneLen(rune(u))
		utf8.EncodeRune(buf[i:], rune(u))
		i--
		buf[i] = '\''
		i--
		buf[i] = ' '
	}
	for u >= 16 {
		i--
		buf[i] = udigits[u&0xF]
		prec--
		u >>= 4
	}
	i--
	buf[i] = udigits[u]
	prec--
	for prec > 0 {
		i--
		buf[i] = '0'
		prec--
	}
	i--
	buf[i] = '+'
	i--
	buf[i] = 'U'
	oldZero := f.zero
	f.zero = false
	f.pad(buf[i:])
	f.zero = oldZero
}

// fmtInteger formats signed and unsigned integers.
func (f *formatter) fmtInteger(u uint64, base int, isSigned bool, verb rune, digits string) {
	negative := isSigned && int64(u) < 0
	if negative {
		u = -u
	}
	width := 68
	if f.widPresent || f.precPresent {
		if w := 3 + f.wid + f.prec; w > width {
			width = w
		}
	}
	buf := make([]byte, width)
	prec := 0
	if f.precPresent {
		prec = f.prec
		if prec == 0 && u == 0 {
			oldZero := f.zero
			f.zero = false
			f.writePadding(f.wid)
			f.zero = oldZero
			return
		}
	} else if f.zero && !f.minus && f.widPresent {
		prec = f.wid
		if negative || f.plus || f.space {
			prec--
		}
	}
	i := len(buf)
	switch base {
	case 10:
		for u >= 10 {
			i--
			next := u / 10
			buf[i] = byte('0' + u - next*10)
			u = next
		}
	case 16:
		for u >= 16 {
			i--
			buf[i] = digits[u&0xF]
			u >>= 4
		}
	case 8:
		for u >= 8 {
			i--
			buf[i] = byte('0' + u&7)
			u >>= 3
		}
	case 2:
		for u >= 2 {
			i--
			buf[i] = byte('0' + u&1)
			u >>= 1
		}
	default:
		panic("fmt: unknown base; can't happen")
	}
	i--
	buf[i] = digits[u]
	for i > 0 && prec > len(buf)-i {
		i--
		buf[i] = '0'
	}
	if f.sharp {
		switch base {
		case 2:
			i--
			buf[i] = 'b'
			i--
			buf[i] = '0'
		case 8:
			if buf[i] != '0' {
				i--
				buf[i] = '0'
			}
		case 16:
			i--
			buf[i] = digits[16]
			i--
			buf[i] = '0'
		}
	}
	if verb == 'O' {
		i--
		buf[i] = 'o'
		i--
		buf[i] = '0'
	}
	if negative {
		i--
		buf[i] = '-'
	} else if f.plus {
		i--
		buf[i] = '+'
	} else if f.space {
		i--
		buf[i] = ' '
	}
	oldZero := f.zero
	f.zero = false
	f.pad(buf[i:])
	f.zero = oldZero
}

func (f *formatter) truncateString(s string) string {
	if f.precPresent {
		n := f.prec
		for i := range s {
			n--
			if n < 0 {
				return s[:i]
			}
		}
	}
	return s
}

func (f *formatter) truncate(b []byte) []byte {
	if f.precPresent {
		n := f.prec
		for i := 0; i < len(b); {
			n--
			if n < 0 {
				return b[:i]
			}
			_, wid := utf8.DecodeRune(b[i:])
			i += wid
		}
	}
	return b
}

func (f *formatter) fmtS(s string) {
	s = f.truncateString(s)
	f.padString(s)
}

func (f *formatter) fmtBs(b []byte) {
	b = f.truncate(b)
	f.pad(b)
}

// fmtSbx formats a string or byte slice as a hexadecimal encoding of its bytes.
func (f *formatter) fmtSbx(s string, b []byte, digits string) {
	length := len(b)
	if b == nil {
		length = len(s)
	}
	if f.precPresent && f.prec < length {
		length = f.prec
	}
	width := 2 * length
	if width > 0 {
		if f.space {
			if f.sharp {
				width *= 2
			}
			width += length - 1
		} else if f.sharp {
			width += 2
		}
	} else {
		if f.widPresent {
			f.writePadding(f.wid)
		}
		return
	}
	if f.widPresent && f.wid > width && !f.minus {
		f.writePadding(f.wid - width)
	}
	if f.sharp {
		f.buf.writeByte('0')
		f.buf.writeByte(digits[16])
	}
	var c byte
	for i := 0; i < length; i++ {
		if f.space && i > 0 {
			f.buf.writeByte(' ')
			if f.sharp {
				f.buf.writeByte('0')
				f.buf.writeByte(digits[16])
			}
		}
		if b != nil {
			c = b[i]
		} else {
			c = s[i]
		}
		f.buf.writeByte(digits[c>>4])
		f.buf.writeByte(digits[c&0xF])
	}
	if f.widPresent && f.wid > width && f.minus {
		f.writePadding(f.wid - width)
	}
}

func (f *formatter) fmtSx(s, digits string) {
	f.fmtSbx(s, nil, digits)
}

func (f *formatter) fmtBx(b []byte, digits string) {
	f.fmtSbx("", b, digits)
}

// fmtQ formats a string as a double-quoted, escaped Go string constant. If
// f.sharp is set a raw (backquoted) string may be returned instead if the
// string does not contain any control characters other than tab.
func (f *formatter) fmtQ(s string) {
	s = f.truncateString(s)
	if f.sharp && strconv.CanBackquote(s) {
		f.padString("`" + s + "`")
		return
	}
	if f.plus {
		f.pad(strconv.AppendQuoteToASCII(nil, s))
	} else {
		f.pad(strconv.AppendQuote(nil, s))
	}
}

// fmtC formats an integer as a Unicode character. If the character is not
// valid Unicode, it will print '�'.
func (f *formatter) fmtC(c uint64) {
	r := rune(c)
	if c > utf8.MaxRune {
		r = utf8.RuneError
	}
	f.pad(utf8.AppendRune(nil, r))
}

// fmtQc formats an integer as a single-quoted, escaped Go character constant.
func (f *formatter) fmtQc(c uint64) {
	r := rune(c)
	if c > utf8.MaxRune {
		r = utf8.RuneError
	}
	if f.plus {
		f.pad(strconv.AppendQuoteRuneToASCII(nil, r))
	} else {
		f.pad(strconv.AppendQuoteRune(nil, r))
	}
}

// fmtFloat formats a float64. It assumes that verb is a valid format
// specifier for strconv.AppendFloat and therefore fits into a byte.
func (f *formatter) fmtFloat(v float64, size int, verb rune, prec int) {
	if f.precPresent {
		prec = f.prec
	}
	num := strconv.AppendFloat([]byte{0}, v, byte(verb), prec, size)
	if num[1] == '-' || num[1] == '+' {
		num = num[1:]
	} else {
		num[0] = '+'
	}
	if f.space && num[0] == '+' && !f.plus {
		num[0] = ' '
	}
	if num[1] == 'I' || num[1] == 'N' {
		oldZero := f.zero
		f.zero = false
		if num[1] == 'N' && !f.space && !f.plus {
			num = num[1:]
		}
		f.pad(num)
		f.zero = oldZero
		return
	}
	if f.sharp && verb != 'b' {
		digits := 0
		switch verb {
		case 'v', 'g', 'G', 'x':
			digits = prec
			if digits == -1 {
				digits = 6
			}
		}
		var tail []byte
		hasDecimalPoint := false
		sawNonzeroDigit := false
	scan:
		for i := 1; i < len(num); i++ {
			switch num[i] {
			case '.':
				hasDecimalPoint = true
			case 'p', 'P':
				tail = append(tail, num[i:]...)
				num = num[:i]
				break scan
			case 'e', 'E':
				if verb != 'x' && verb != 'X' {
					tail = append(tail, num[i:]...)
					num = num[:i]
					break scan
				}
				if num[i] != '0' {
					sawNonzeroDigit = true
				}
				if sawNonzeroDigit {
					digits--
				}
			default:
				if num[i] != '0' {
					sawNonzeroDigit = true
				}
				if sawNonzeroDigit {
					digits--
				}
			}
		}
		if !hasDecimalPoint {
			if len(num) == 2 && num[1] == '0' {
				digits--
			}
			num = append(num, '.')
		}
		for digits > 0 {
			num = append(num, '0')
			digits--
		}
		num = append(num, tail...)
	}
	if f.plus || num[0] != '+' {
		if f.zero && !f.minus && f.widPresent && f.wid > len(num) {
			f.buf.writeByte(num[0])
			f.writePadding(f.wid - len(num))
			f.buf.write(num[1:])
			return
		}
		f.pad(num)
		return
	}
	f.pad(num[1:])
}
