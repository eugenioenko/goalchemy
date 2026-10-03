// Package bytes manipulates byte slices. Functions that return parts of
// their input return subslices that share its storage, as in Go.
package bytes

import (
	"github.com/eugenioenko/goalchemy/lib/errors"
	"github.com/eugenioenko/goalchemy/std/strings"
	"github.com/eugenioenko/goalchemy/std/unicode"
	"github.com/eugenioenko/goalchemy/std/unicode/utf8"
)

// Equal reports whether a and b have the same bytes; nil equals empty.
func Equal(a, b []byte) bool { return string(a) == string(b) }

// Compare returns 0 if a == b, -1 if a < b, and +1 if a > b.
func Compare(a, b []byte) int { return strings.Compare(string(a), string(b)) }

// Index returns the index of the first instance of sep in s, or -1.
func Index(s, sep []byte) int { return strings.Index(string(s), string(sep)) }

// LastIndex returns the index of the last instance of sep in s, or -1.
func LastIndex(s, sep []byte) int { return strings.LastIndex(string(s), string(sep)) }

// IndexByte returns the index of the first instance of c in b, or -1.
func IndexByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}

// IndexRune returns the index of the first instance of r in s, or -1.
func IndexRune(s []byte, r rune) int { return strings.IndexRune(string(s), r) }

// Contains reports whether sub is within b.
func Contains(b, sub []byte) bool { return Index(b, sub) >= 0 }

// ContainsRune reports whether r is within b.
func ContainsRune(b []byte, r rune) bool { return IndexRune(b, r) >= 0 }

// Count counts the non-overlapping instances of sep in s.
func Count(s, sep []byte) int { return strings.Count(string(s), string(sep)) }

// HasPrefix reports whether s begins with prefix.
func HasPrefix(s, prefix []byte) bool { return len(s) >= len(prefix) && Equal(s[:len(prefix)], prefix) }

// HasSuffix reports whether s ends with suffix.
func HasSuffix(s, suffix []byte) bool {
	return len(s) >= len(suffix) && Equal(s[len(s)-len(suffix):], suffix)
}

// TrimPrefix returns s without prefix if present.
func TrimPrefix(s, prefix []byte) []byte {
	if HasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

// TrimSuffix returns s without suffix if present.
func TrimSuffix(s, suffix []byte) []byte {
	if HasSuffix(s, suffix) {
		return s[:len(s)-len(suffix)]
	}
	return s
}

// TrimSpace returns the subslice of s without leading and trailing white space.
func TrimSpace(s []byte) []byte {
	start := 0
	for start < len(s) {
		r, size := utf8.DecodeRune(s[start:])
		if !unicode.IsSpace(r) {
			break
		}
		start += size
	}
	end := len(s)
	for end > start {
		r, size := utf8.DecodeLastRune(s[start:end])
		if !unicode.IsSpace(r) {
			break
		}
		end -= size
	}
	if start == end {
		return nil
	}
	return s[start:end]
}

// Fields splits s around runs of white space into subslices of s.
func Fields(s []byte) [][]byte {
	out := [][]byte{}
	start := -1
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRune(s[i:])
		if unicode.IsSpace(r) {
			if start >= 0 {
				out = append(out, s[start:i:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
		i += size
	}
	if start >= 0 {
		out = append(out, s[start:len(s):len(s)])
	}
	return out
}

// Split slices s into all subslices separated by sep.
func Split(s, sep []byte) [][]byte { return SplitN(s, sep, -1) }

// SplitN slices s into at most n subslices separated by sep; n < 0 means all.
func SplitN(s, sep []byte, n int) [][]byte {
	if n == 0 {
		return nil
	}
	if len(sep) == 0 {
		l := utf8.RuneCount(s)
		if n < 0 || n > l {
			n = l
		}
		a := make([][]byte, n)
		for i := 0; i < n-1; i++ {
			_, size := utf8.DecodeRune(s)
			a[i] = s[:size:size]
			s = s[size:]
		}
		if n > 0 {
			a[n-1] = s
		}
		return a
	}
	if n < 0 {
		n = Count(s, sep) + 1
	}
	if n > len(s)+1 {
		n = len(s) + 1
	}
	a := make([][]byte, n)
	n--
	i := 0
	for i < n {
		m := Index(s, sep)
		if m < 0 {
			break
		}
		a[i] = s[:m:m]
		s = s[m+len(sep):]
		i++
	}
	a[i] = s
	return a[:i+1]
}

// Join concatenates s with sep between elements into a new slice.
func Join(s [][]byte, sep []byte) []byte {
	if len(s) == 0 {
		return []byte{}
	}
	if len(s) == 1 {
		return append([]byte(nil), s[0]...)
	}
	var out []byte
	for i, b := range s {
		if i > 0 {
			out = append(out, sep...)
		}
		out = append(out, b...)
	}
	if out == nil {
		out = []byte{}
	}
	return out
}

// Repeat returns count copies of b. It panics if count is negative.
func Repeat(b []byte, count int) []byte {
	if count < 0 {
		panic("bytes: negative Repeat count")
	}
	out := make([]byte, 0, len(b)*count)
	for i := 0; i < count; i++ {
		out = append(out, b...)
	}
	return out
}

// ToUpper returns a copy of s with every rune mapped to upper case.
func ToUpper(s []byte) []byte { return []byte(strings.ToUpper(string(s))) }

// ToLower returns a copy of s with every rune mapped to lower case.
func ToLower(s []byte) []byte { return []byte(strings.ToLower(string(s))) }

// EqualFold reports whether s and t are equal under simple Unicode case folding.
func EqualFold(s, t []byte) bool { return strings.EqualFold(string(s), string(t)) }

// Clone returns a copy of b, or nil if b is nil.
func Clone(b []byte) []byte {
	if b == nil {
		return nil
	}
	return append([]byte{}, b...)
}

// ErrTooLarge is returned when a Buffer cannot grow.
var ErrTooLarge = errors.New("bytes.Buffer: too large")

// Buffer is a growable byte buffer. The zero value is an empty buffer.
type Buffer struct {
	buf []byte
	off int
}

// NewBuffer returns a Buffer whose initial contents are buf.
func NewBuffer(buf []byte) *Buffer { return &Buffer{buf: buf} }

// NewBufferString returns a Buffer whose initial contents are s.
func NewBufferString(s string) *Buffer { return &Buffer{buf: []byte(s)} }

// Bytes returns the unread portion of the buffer, sharing its storage.
func (b *Buffer) Bytes() []byte { return b.buf[b.off:] }

// String returns the unread portion of the buffer as a string.
func (b *Buffer) String() string {
	if b == nil {
		return "<nil>"
	}
	return string(b.buf[b.off:])
}

// Len returns the number of unread bytes.
func (b *Buffer) Len() int { return len(b.buf) - b.off }

// Reset empties the buffer.
func (b *Buffer) Reset() {
	b.buf = b.buf[:0]
	b.off = 0
}

// Truncate keeps the first n unread bytes. It panics if n is out of range.
func (b *Buffer) Truncate(n int) {
	if n == 0 {
		b.Reset()
		return
	}
	if n < 0 || n > b.Len() {
		panic("bytes.Buffer: truncation out of range")
	}
	b.buf = b.buf[:b.off+n]
}

// Grow ensures space for another n bytes. It panics if n is negative.
func (b *Buffer) Grow(n int) {
	if n < 0 {
		panic("bytes.Buffer.Grow: negative count")
	}
	if cap(b.buf)-len(b.buf) < n {
		buf := make([]byte, len(b.buf), 2*cap(b.buf)+n)
		copy(buf, b.buf)
		b.buf = buf
	}
}

// Write appends p and always returns len(p), nil.
func (b *Buffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

// WriteString appends s and always returns len(s), nil.
func (b *Buffer) WriteString(s string) (int, error) {
	b.buf = append(b.buf, s...)
	return len(s), nil
}

// WriteByte appends c and always returns nil.
func (b *Buffer) WriteByte(c byte) error {
	b.buf = append(b.buf, c)
	return nil
}

// WriteRune appends the UTF-8 encoding of r and returns its length.
func (b *Buffer) WriteRune(r rune) (int, error) {
	n := len(b.buf)
	b.buf = utf8.AppendRune(b.buf, r)
	return len(b.buf) - n, nil
}

// Next returns the next n unread bytes, or all of them if fewer remain, and
// advances past them.
func (b *Buffer) Next(n int) []byte {
	m := b.Len()
	if n > m {
		n = m
	}
	data := b.buf[b.off : b.off+n]
	b.off += n
	return data
}

// ReadByte returns the next byte, or an error when the buffer is empty.
func (b *Buffer) ReadByte() (byte, error) {
	if b.off >= len(b.buf) {
		b.Reset()
		return 0, errEOF
	}
	c := b.buf[b.off]
	b.off++
	return c, nil
}

var errEOF = errors.New("EOF")
