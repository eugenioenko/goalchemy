// Package bytes manipulates byte slices. Functions that return parts of
// their input return subslices that share its storage, as in Go.
package bytes

import (
	"github.com/eugenioenko/goalchemy/lib/errors"
	"github.com/eugenioenko/goalchemy/std/io"
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

// ReadByte returns the next byte, or io.EOF when the buffer is empty.
func (b *Buffer) ReadByte() (byte, error) {
	if b.off >= len(b.buf) {
		b.Reset()
		return 0, io.EOF
	}
	c := b.buf[b.off]
	b.off++
	return c, nil
}

// Read reads the next len(p) bytes from the buffer, or until it is drained. It
// returns io.EOF when the buffer has no data and p is not empty.
func (b *Buffer) Read(p []byte) (int, error) {
	if b.off >= len(b.buf) {
		b.Reset()
		if len(p) == 0 {
			return 0, nil
		}
		return 0, io.EOF
	}
	n := copy(p, b.buf[b.off:])
	b.off += n
	return n, nil
}

// ReadFrom appends data from r until io.EOF and returns the number of bytes
// read. Errors other than io.EOF are returned.
func (b *Buffer) ReadFrom(r io.Reader) (int64, error) {
	var total int64
	for {
		b.Grow(512)
		n, err := r.Read(b.buf[len(b.buf):cap(b.buf)])
		if n < 0 {
			panic("bytes.Buffer: reader returned negative count from Read")
		}
		b.buf = b.buf[:len(b.buf)+n]
		total += int64(n)
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}

// WriteTo writes the unread data to w until the buffer is drained or an error
// occurs.
func (b *Buffer) WriteTo(w io.Writer) (int64, error) {
	var total int64
	if n := b.Len(); n > 0 {
		m, err := w.Write(b.buf[b.off:])
		if m > n {
			panic("bytes.Buffer.WriteTo: invalid Write count")
		}
		b.off += m
		total = int64(m)
		if err != nil {
			return total, err
		}
		if m != n {
			return total, io.ErrShortWrite
		}
	}
	b.Reset()
	return total, nil
}

var (
	errNegativePosition = errors.New("bytes.Reader.Seek: negative position")
	errNegativeOffset   = errors.New("bytes.Reader.ReadAt: negative offset")
	errInvalidWhence    = errors.New("bytes.Reader.Seek: invalid whence")
	errUnreadByte       = errors.New("bytes.Reader.UnreadByte: at beginning of slice")
	errUnreadRune       = errors.New("bytes.Reader.UnreadRune: previous operation was not ReadRune")
	errUnreadRuneStart  = errors.New("bytes.Reader.UnreadRune: at beginning of slice")
)

// Reader implements io.Reader, io.ReaderAt, io.Seeker, io.WriterTo,
// io.ByteScanner and io.RuneScanner over a byte slice. It does not modify the
// slice. The zero value reads nothing.
type Reader struct {
	s        []byte
	i        int64
	prevRune int
}

// NewReader returns a Reader reading from b.
func NewReader(b []byte) *Reader { return &Reader{b, 0, -1} }

// Len returns the number of unread bytes.
func (r *Reader) Len() int {
	if r.i >= int64(len(r.s)) {
		return 0
	}
	return int(int64(len(r.s)) - r.i)
}

// Size returns the original length of the underlying slice.
func (r *Reader) Size() int64 { return int64(len(r.s)) }

// Read implements io.Reader.
func (r *Reader) Read(b []byte) (n int, err error) {
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	r.prevRune = -1
	n = copy(b, r.s[r.i:])
	r.i += int64(n)
	return n, nil
}

// ReadAt implements io.ReaderAt. It does not change the read position.
func (r *Reader) ReadAt(b []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, errNegativeOffset
	}
	if off >= int64(len(r.s)) {
		return 0, io.EOF
	}
	n = copy(b, r.s[off:])
	if n < len(b) {
		err = io.EOF
	}
	return n, err
}

// ReadByte implements io.ByteReader.
func (r *Reader) ReadByte() (byte, error) {
	r.prevRune = -1
	if r.i >= int64(len(r.s)) {
		return 0, io.EOF
	}
	b := r.s[r.i]
	r.i++
	return b, nil
}

// UnreadByte steps back one byte.
func (r *Reader) UnreadByte() error {
	if r.i <= 0 {
		return errUnreadByte
	}
	r.prevRune = -1
	r.i--
	return nil
}

// ReadRune implements io.RuneReader.
func (r *Reader) ReadRune() (ch rune, size int, err error) {
	if r.i >= int64(len(r.s)) {
		r.prevRune = -1
		return 0, 0, io.EOF
	}
	r.prevRune = int(r.i)
	if c := r.s[r.i]; c < utf8.RuneSelf {
		r.i++
		return rune(c), 1, nil
	}
	ch, size = utf8.DecodeRune(r.s[r.i:])
	r.i += int64(size)
	return ch, size, nil
}

// UnreadRune steps back over the rune returned by the previous ReadRune.
func (r *Reader) UnreadRune() error {
	if r.i <= 0 {
		return errUnreadRuneStart
	}
	if r.prevRune < 0 {
		return errUnreadRune
	}
	r.i = int64(r.prevRune)
	r.prevRune = -1
	return nil
}

// Seek implements io.Seeker. Seeking past the end is allowed.
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	r.prevRune = -1
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.i + offset
	case io.SeekEnd:
		abs = int64(len(r.s)) + offset
	default:
		return 0, errInvalidWhence
	}
	if abs < 0 {
		return 0, errNegativePosition
	}
	r.i = abs
	return abs, nil
}

// WriteTo implements io.WriterTo, writing the unread data to w.
func (r *Reader) WriteTo(w io.Writer) (n int64, err error) {
	r.prevRune = -1
	if r.i >= int64(len(r.s)) {
		return 0, nil
	}
	b := r.s[r.i:]
	m, err := w.Write(b)
	if m > len(b) {
		panic("bytes.Reader.WriteTo: invalid Write count")
	}
	r.i += int64(m)
	n = int64(m)
	if m != len(b) && err == nil {
		err = io.ErrShortWrite
	}
	return n, err
}

// Reset resets the Reader to read from b.
func (r *Reader) Reset(b []byte) { *r = Reader{b, 0, -1} }
