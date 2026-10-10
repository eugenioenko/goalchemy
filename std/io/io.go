// Package io provides basic interfaces to I/O primitives and helpers that
// combine them, matching Go's io package. Pipe is not provided.
package io

import "github.com/eugenioenko/goalchemy/lib/errors"

// Seek whence values.
const (
	SeekStart   = 0
	SeekCurrent = 1
	SeekEnd     = 2
)

// EOF is the error returned by Read when no more input is available.
var EOF = errors.New("EOF")

// ErrUnexpectedEOF means that EOF was encountered in the middle of reading a
// fixed-size block or data structure.
var ErrUnexpectedEOF = errors.New("unexpected EOF")

// ErrShortWrite means that a write accepted fewer bytes than requested but
// failed to return an explicit error.
var ErrShortWrite = errors.New("short write")

// ErrShortBuffer means that a read required a longer buffer than was provided.
var ErrShortBuffer = errors.New("short buffer")

// ErrNoProgress is returned when many calls to Read have failed to return any
// data or error.
var ErrNoProgress = errors.New("multiple Read calls return no data or error")

// ErrClosedPipe is the error used for read or write operations on a closed pipe.
var ErrClosedPipe = errors.New("io: read/write on closed pipe")

var errInvalidWrite = errors.New("invalid write result")
var errWhence = errors.New("Seek: invalid whence")
var errOffset = errors.New("Seek: invalid offset")

// Reader wraps the basic Read method.
type Reader interface {
	Read(p []byte) (n int, err error)
}

// Writer wraps the basic Write method.
type Writer interface {
	Write(p []byte) (n int, err error)
}

// Closer wraps the basic Close method.
type Closer interface {
	Close() error
}

// Seeker wraps the basic Seek method.
type Seeker interface {
	Seek(offset int64, whence int) (int64, error)
}

// ReadWriter groups the basic Read and Write methods.
type ReadWriter interface {
	Reader
	Writer
}

// ReadCloser groups the basic Read and Close methods.
type ReadCloser interface {
	Reader
	Closer
}

// WriteCloser groups the basic Write and Close methods.
type WriteCloser interface {
	Writer
	Closer
}

// ReadWriteCloser groups the basic Read, Write and Close methods.
type ReadWriteCloser interface {
	Reader
	Writer
	Closer
}

// ReadSeeker groups the basic Read and Seek methods.
type ReadSeeker interface {
	Reader
	Seeker
}

// ReadSeekCloser groups the basic Read, Seek and Close methods.
type ReadSeekCloser interface {
	Reader
	Seeker
	Closer
}

// WriteSeeker groups the basic Write and Seek methods.
type WriteSeeker interface {
	Writer
	Seeker
}

// ReadWriteSeeker groups the basic Read, Write and Seek methods.
type ReadWriteSeeker interface {
	Reader
	Writer
	Seeker
}

// ReaderFrom wraps the ReadFrom method.
type ReaderFrom interface {
	ReadFrom(r Reader) (n int64, err error)
}

// WriterTo wraps the WriteTo method.
type WriterTo interface {
	WriteTo(w Writer) (n int64, err error)
}

// ReaderAt wraps the basic ReadAt method.
type ReaderAt interface {
	ReadAt(p []byte, off int64) (n int, err error)
}

// WriterAt wraps the basic WriteAt method.
type WriterAt interface {
	WriteAt(p []byte, off int64) (n int, err error)
}

// ByteReader wraps the ReadByte method.
type ByteReader interface {
	ReadByte() (byte, error)
}

// ByteScanner adds the UnreadByte method to ByteReader.
type ByteScanner interface {
	ByteReader
	UnreadByte() error
}

// ByteWriter wraps the WriteByte method.
type ByteWriter interface {
	WriteByte(c byte) error
}

// RuneReader wraps the ReadRune method.
type RuneReader interface {
	ReadRune() (r rune, size int, err error)
}

// RuneScanner adds the UnreadRune method to RuneReader.
type RuneScanner interface {
	RuneReader
	UnreadRune() error
}

// StringWriter wraps the WriteString method.
type StringWriter interface {
	WriteString(s string) (n int, err error)
}

// WriteString writes s to w, using w's WriteString method when it has one.
func WriteString(w Writer, s string) (n int, err error) {
	if sw, ok := w.(StringWriter); ok {
		return sw.WriteString(s)
	}
	return w.Write([]byte(s))
}

// ReadAtLeast reads from r into buf until it has read at least min bytes. It
// returns ErrShortBuffer if min is larger than buf, EOF if no bytes were read,
// and ErrUnexpectedEOF if EOF happens after reading fewer than min bytes.
func ReadAtLeast(r Reader, buf []byte, min int) (n int, err error) {
	if len(buf) < min {
		return 0, ErrShortBuffer
	}
	for n < min && err == nil {
		var nn int
		nn, err = r.Read(buf[n:])
		n += nn
	}
	if n >= min {
		err = nil
	} else if n > 0 && err == EOF {
		err = ErrUnexpectedEOF
	}
	return n, err
}

// ReadFull reads exactly len(buf) bytes from r into buf.
func ReadFull(r Reader, buf []byte) (n int, err error) {
	return ReadAtLeast(r, buf, len(buf))
}

// CopyN copies n bytes, or until an error, from src to dst.
func CopyN(dst Writer, src Reader, n int64) (written int64, err error) {
	written, err = Copy(dst, LimitReader(src, n))
	if written == n {
		return n, nil
	}
	if written < n && err == nil {
		err = EOF
	}
	return written, err
}

// Copy copies from src to dst until EOF or an error. A successful Copy
// returns err == nil, not EOF.
func Copy(dst Writer, src Reader) (written int64, err error) {
	return copyBuffer(dst, src, nil)
}

// CopyBuffer is identical to Copy except that it stages through buf. It
// panics if buf is non-nil and empty.
func CopyBuffer(dst Writer, src Reader, buf []byte) (written int64, err error) {
	if buf != nil && len(buf) == 0 {
		panic("empty buffer in CopyBuffer")
	}
	return copyBuffer(dst, src, buf)
}

func copyBuffer(dst Writer, src Reader, buf []byte) (written int64, err error) {
	if wt, ok := src.(WriterTo); ok {
		return wt.WriteTo(dst)
	}
	if rf, ok := dst.(ReaderFrom); ok {
		return rf.ReadFrom(src)
	}
	if buf == nil {
		size := 32 * 1024
		if l, ok := src.(*LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf = make([]byte, size)
	}
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errInvalidWrite
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

// ReadAll reads from r until an error or EOF and returns the data it read. A
// successful call returns err == nil, not EOF.
func ReadAll(r Reader) ([]byte, error) {
	b := make([]byte, 0, 512)
	for {
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if err == EOF {
				err = nil
			}
			return b, err
		}
		if len(b) == cap(b) {
			b = append(b, 0)[:len(b)]
		}
	}
}

// LimitReader returns a Reader that reads from r but stops with EOF after n
// bytes.
func LimitReader(r Reader, n int64) Reader { return &LimitedReader{r, n} }

// LimitedReader reads from R but limits the amount of data returned to just N
// bytes.
type LimitedReader struct {
	R Reader
	N int64
}

// Read reads at most the remaining N bytes.
func (l *LimitedReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, EOF
	}
	if int64(len(p)) > l.N {
		p = p[0:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	return n, err
}

// NewSectionReader returns a SectionReader that reads from r starting at
// offset off and stops with EOF after n bytes.
func NewSectionReader(r ReaderAt, off int64, n int64) *SectionReader {
	var remaining int64
	const maxint64 = 1<<63 - 1
	if off <= maxint64-n {
		remaining = n + off
	} else {
		remaining = maxint64
	}
	return &SectionReader{r, off, off, remaining, n}
}

// SectionReader implements Read, Seek and ReadAt on a section of an underlying
// ReaderAt.
type SectionReader struct {
	r     ReaderAt
	base  int64
	off   int64
	limit int64
	n     int64
}

// Read implements io.Reader within the section.
func (s *SectionReader) Read(p []byte) (n int, err error) {
	if s.off >= s.limit {
		return 0, EOF
	}
	if rest := s.limit - s.off; int64(len(p)) > rest {
		p = p[0:rest]
	}
	n, err = s.r.ReadAt(p, s.off)
	s.off += int64(n)
	return n, err
}

// Seek implements io.Seeker relative to the section.
func (s *SectionReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	default:
		return 0, errWhence
	case SeekStart:
		offset += s.base
	case SeekCurrent:
		offset += s.off
	case SeekEnd:
		offset += s.limit
	}
	if offset < s.base {
		return 0, errOffset
	}
	s.off = offset
	return offset - s.base, nil
}

// ReadAt implements io.ReaderAt relative to the section.
func (s *SectionReader) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= s.Size() {
		return 0, EOF
	}
	off += s.base
	if rest := s.limit - off; int64(len(p)) > rest {
		p = p[0:rest]
		n, err = s.r.ReadAt(p, off)
		if err == nil {
			err = EOF
		}
		return n, err
	}
	return s.r.ReadAt(p, off)
}

// Size returns the size of the section in bytes.
func (s *SectionReader) Size() int64 { return s.limit - s.base }

// Outer returns the underlying ReaderAt and offsets of the section.
func (s *SectionReader) Outer() (r ReaderAt, off int64, n int64) {
	return s.r, s.base, s.n
}

// OffsetWriter maps writes at offset base to offset base+off in the
// underlying WriterAt.
type OffsetWriter struct {
	w    WriterAt
	base int64
	off  int64
}

// NewOffsetWriter returns an OffsetWriter that writes to w starting at off.
func NewOffsetWriter(w WriterAt, off int64) *OffsetWriter {
	return &OffsetWriter{w, off, off}
}

// Write writes at the current offset and advances it.
func (o *OffsetWriter) Write(p []byte) (n int, err error) {
	n, err = o.w.WriteAt(p, o.off)
	o.off += int64(n)
	return n, err
}

// WriteAt writes at off relative to the base offset.
func (o *OffsetWriter) WriteAt(p []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, errOffset
	}
	return o.w.WriteAt(p, off+o.base)
}

// Seek implements io.Seeker relative to the base offset; SeekEnd is not supported.
func (o *OffsetWriter) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	default:
		return 0, errWhence
	case SeekStart:
		offset += o.base
	case SeekCurrent:
		offset += o.off
	}
	if offset < o.base {
		return 0, errOffset
	}
	o.off = offset
	return offset - o.base, nil
}

// TeeReader returns a Reader that writes to w what it reads from r. Any error
// writing to w is returned as a read error.
func TeeReader(r Reader, w Writer) Reader { return &teeReader{r, w} }

type teeReader struct {
	r Reader
	w Writer
}

func (t *teeReader) Read(p []byte) (n int, err error) {
	n, err = t.r.Read(p)
	if n > 0 {
		if n, err := t.w.Write(p[:n]); err != nil {
			return n, err
		}
	}
	return n, err
}

// Discard is a Writer on which all Write calls succeed without doing anything.
var Discard Writer = discard{}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

func (discard) WriteString(s string) (int, error) { return len(s), nil }

func (discard) ReadFrom(r Reader) (n int64, err error) {
	buf := make([]byte, 8192)
	for {
		readSize, err := r.Read(buf)
		n += int64(readSize)
		if err != nil {
			if err == EOF {
				return n, nil
			}
			return n, err
		}
	}
}

// NopCloser returns a ReadCloser with a no-op Close method wrapping r. If r
// implements WriterTo, the result does too.
func NopCloser(r Reader) ReadCloser {
	if _, ok := r.(WriterTo); ok {
		return nopCloserWriterTo{r}
	}
	return nopCloser{r}
}

type nopCloser struct{ Reader }

func (nopCloser) Close() error { return nil }

type nopCloserWriterTo struct{ Reader }

func (nopCloserWriterTo) Close() error { return nil }

func (c nopCloserWriterTo) WriteTo(w Writer) (n int64, err error) {
	return c.Reader.(WriterTo).WriteTo(w)
}

type eofReader struct{}

func (eofReader) Read([]byte) (int, error) { return 0, EOF }

type multiReader struct {
	readers []Reader
}

func (mr *multiReader) Read(p []byte) (n int, err error) {
	for len(mr.readers) > 0 {
		if len(mr.readers) == 1 {
			if r, ok := mr.readers[0].(*multiReader); ok {
				mr.readers = r.readers
				continue
			}
		}
		n, err = mr.readers[0].Read(p)
		if err == EOF {
			mr.readers[0] = eofReader{}
			mr.readers = mr.readers[1:]
		}
		if n > 0 || err != EOF {
			if err == EOF && len(mr.readers) > 0 {
				err = nil
			}
			return
		}
	}
	return 0, EOF
}

// MultiReader returns a Reader that is the logical concatenation of readers.
func MultiReader(readers ...Reader) Reader {
	r := make([]Reader, len(readers))
	copy(r, readers)
	return &multiReader{r}
}

type multiWriter struct {
	writers []Writer
}

func (t *multiWriter) Write(p []byte) (n int, err error) {
	for _, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			err = ErrShortWrite
			return
		}
	}
	return len(p), nil
}

func (t *multiWriter) WriteString(s string) (n int, err error) {
	var p []byte
	for _, w := range t.writers {
		if sw, ok := w.(StringWriter); ok {
			n, err = sw.WriteString(s)
		} else {
			if p == nil {
				p = []byte(s)
			}
			n, err = w.Write(p)
		}
		if err != nil {
			return
		}
		if n != len(s) {
			err = ErrShortWrite
			return
		}
	}
	return len(s), nil
}

// MultiWriter creates a writer that duplicates its writes to all the provided
// writers, stopping at the first error.
func MultiWriter(writers ...Writer) Writer {
	all := make([]Writer, 0, len(writers))
	for _, w := range writers {
		if mw, ok := w.(*multiWriter); ok {
			all = append(all, mw.writers...)
		} else {
			all = append(all, w)
		}
	}
	return &multiWriter{all}
}
