// Package unicode classifies runes and maps their case using the Unicode
// tables of the reference Go toolchain.
package unicode

const (
	MaxRune         = '\U0010FFFF'
	ReplacementChar = '�'
	MaxASCII        = '\u007F'
	MaxLatin1       = 'ÿ'
)

const upperLower = 0x7FFFFF

func at(table string, i int) rune {
	return rune(table[3*i])<<16 | rune(table[3*i+1])<<8 | rune(table[3*i+2])
}

func in(table string, r rune) bool {
	n := len(table) / 6
	lo, hi := 0, n
	for lo < hi {
		m := lo + (hi-lo)/2
		if at(table, 2*m+1) < r {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return lo < n && at(table, 2*lo) <= r
}

// IsLetter reports whether r is a letter (category L).
func IsLetter(r rune) bool { return in(letterTable, r) }

// IsUpper reports whether r is an upper case letter.
func IsUpper(r rune) bool { return in(upperTable, r) }

// IsLower reports whether r is a lower case letter.
func IsLower(r rune) bool { return in(lowerTable, r) }

// IsTitle reports whether r is a title case letter.
func IsTitle(r rune) bool { return in(titleTable, r) }

// IsDigit reports whether r is a decimal digit (category Nd).
func IsDigit(r rune) bool {
	if uint32(r) <= MaxLatin1 {
		return '0' <= r && r <= '9'
	}
	return in(digitTable, r)
}

// IsNumber reports whether r is a number (category N).
func IsNumber(r rune) bool { return in(numberTable, r) }

// IsPunct reports whether r is punctuation (category P).
func IsPunct(r rune) bool { return in(punctTable, r) }

// IsSymbol reports whether r is a symbol (category S).
func IsSymbol(r rune) bool { return in(symbolTable, r) }

// IsMark reports whether r is a mark (category M).
func IsMark(r rune) bool { return in(markTable, r) }

// IsControl reports whether r is a control character.
func IsControl(r rune) bool { return in(controlTable, r) }

// IsPrint reports whether r is printable as defined by Go: letters, marks,
// numbers, punctuation, symbols, and the ASCII space.
func IsPrint(r rune) bool { return in(printTable, r) }

// IsGraphic reports whether r is a graphic character, including Unicode
// spaces.
func IsGraphic(r rune) bool { return in(graphicTable, r) }

// IsSpace reports whether r is a white space character as defined by Go.
func IsSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0x85, 0xA0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000:
		return true
	}
	return 0x2000 <= r && r <= 0x200A
}

const (
	upperCase = 0
	lowerCase = 1
	titleCase = 2
)

func to(c int, r rune) rune {
	n := len(caseTable) / 15
	lo, hi := 0, n
	for lo < hi {
		m := lo + (hi-lo)/2
		if at(caseTable, 5*m+1) < r {
			lo = m + 1
		} else {
			hi = m
		}
	}
	if lo == n || at(caseTable, 5*lo) > r {
		return r
	}
	base := at(caseTable, 5*lo)
	var d rune
	switch c {
	case lowerCase:
		d = at(caseTable, 5*lo+2)
	case upperCase:
		d = at(caseTable, 5*lo+3)
	default:
		d = at(caseTable, 5*lo+4)
	}
	if d != upperLower && d >= 0x800000 {
		d -= 0x1000000
	}
	if d == upperLower {
		pair := base + (r-base)&^1
		if c == lowerCase {
			return pair + 1
		}
		return pair
	}
	return r + d
}

// ToUpper maps r to upper case.
func ToUpper(r rune) rune {
	if r < 0x80 {
		if 'a' <= r && r <= 'z' {
			r -= 'a' - 'A'
		}
		return r
	}
	return to(upperCase, r)
}

// ToLower maps r to lower case.
func ToLower(r rune) rune {
	if r < 0x80 {
		if 'A' <= r && r <= 'Z' {
			r += 'a' - 'A'
		}
		return r
	}
	return to(lowerCase, r)
}

// ToTitle maps r to title case.
func ToTitle(r rune) rune {
	if r < 0x80 {
		if 'a' <= r && r <= 'z' {
			r -= 'a' - 'A'
		}
		return r
	}
	return to(titleCase, r)
}

// SimpleFold iterates over the runes equivalent to r under Unicode simple
// case folding, returning the smallest equivalent rune greater than r, or
// the smallest equivalent rune if none is greater.
func SimpleFold(r rune) rune {
	if r < 0 || r > MaxRune {
		return r
	}
	n := len(foldTable) / 6
	lo, hi := 0, n
	for lo < hi {
		m := lo + (hi-lo)/2
		if at(foldTable, 2*m) < r {
			lo = m + 1
		} else {
			hi = m
		}
	}
	if lo < n && at(foldTable, 2*lo) == r {
		return at(foldTable, 2*lo+1)
	}
	if l := ToLower(r); l != r {
		return l
	}
	return ToUpper(r)
}
