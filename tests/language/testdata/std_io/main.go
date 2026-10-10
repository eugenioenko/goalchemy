package main

import (
	"github.com/eugenioenko/goalchemy/std/bytes"
	"github.com/eugenioenko/goalchemy/std/errors"
	"github.com/eugenioenko/goalchemy/std/fmt"
	"github.com/eugenioenko/goalchemy/std/io"
	"github.com/eugenioenko/goalchemy/std/strings"
)

// trickle returns at most step bytes per Read and then fails with fail, or
// io.EOF when fail is nil.
type trickle struct {
	data  []byte
	step  int
	fail  error
	reads int
}

func (t *trickle) Read(p []byte) (int, error) {
	t.reads++
	if len(t.data) == 0 {
		if t.fail != nil {
			return 0, t.fail
		}
		return 0, io.EOF
	}
	n := t.step
	if n > len(p) {
		n = len(p)
	}
	if n > len(t.data) {
		n = len(t.data)
	}
	copy(p, t.data[:n])
	t.data = t.data[n:]
	return n, nil
}

// capped accepts at most max bytes per Write without an error.
type capped struct {
	buf bytes.Buffer
	max int
}

func (c *capped) Write(p []byte) (int, error) {
	if len(p) > c.max {
		p = p[:c.max]
	}
	return c.buf.Write(p)
}

// sparse is a WriterAt over a growable slice.
type sparse struct{ data []byte }

func (s *sparse) WriteAt(p []byte, off int64) (int, error) {
	for int64(len(s.data)) < off+int64(len(p)) {
		s.data = append(s.data, '.')
	}
	return copy(s.data[off:], p), nil
}

type broken struct{}

func (broken) Write([]byte) (int, error) { return 0, errors.New("sink broke") }

var errDisk = errors.New("disk on fire")

func errText(err error) string {
	if err == nil {
		return "nil"
	}
	return err.Error()
}

func src(s string, step int) *trickle { return &trickle{data: []byte(s), step: step} }

func readers() {
	buf := make([]byte, 5)
	n, err := io.ReadFull(src("hello world", 2), buf)
	println(fmt.Sprintf("ReadFull %d %q %s", n, buf[:n], errText(err)))
	n, err = io.ReadFull(src("hey", 2), buf)
	println(fmt.Sprintf("ReadFull short %d %q unexpected=%t", n, buf[:n], err == io.ErrUnexpectedEOF))
	n, err = io.ReadFull(src("", 2), buf)
	println(fmt.Sprintf("ReadFull empty %d eof=%t is=%t", n, err == io.EOF, errors.Is(fmt.Errorf("wrapped: %w", err), io.EOF)))
	n, err = io.ReadAtLeast(src("abcdef", 1), buf, 3)
	println(fmt.Sprintf("ReadAtLeast %d %q %s", n, buf[:n], errText(err)))
	_, err = io.ReadAtLeast(src("abc", 1), buf, 9)
	println("ReadAtLeast short buffer", err == io.ErrShortBuffer)
	n, err = io.ReadFull(&trickle{data: []byte("ab"), step: 1, fail: errDisk}, buf)
	println(fmt.Sprintf("ReadFull failure %d %s", n, errText(err)))

	all, err := io.ReadAll(src(strings.Repeat("xyz", 400), 7))
	println(fmt.Sprintf("ReadAll %d %s", len(all), errText(err)))
	all, err = io.ReadAll(&trickle{data: []byte("partial"), step: 3, fail: errDisk})
	println(fmt.Sprintf("ReadAll failure %q %s", all, errText(err)))
	all, _ = io.ReadAll(io.LimitReader(src("limited reader", 4), 7))
	println(fmt.Sprintf("LimitReader %q", all))
	all, _ = io.ReadAll(io.MultiReader(src("one,", 2), src("", 1), io.MultiReader(src("two,", 3)), strings.NewReader("three")))
	println(fmt.Sprintf("MultiReader %q", all))

	var seen bytes.Buffer
	all, _ = io.ReadAll(io.TeeReader(src("teed", 1), &seen))
	println(fmt.Sprintf("TeeReader %q %q", all, seen.String()))
	_, err = io.TeeReader(src("x", 1), broken{}).Read(buf)
	println("TeeReader error", errText(err))
}

func copies() {
	var out bytes.Buffer
	t := src(strings.Repeat("0123456789", 5000), 999)
	n, err := io.Copy(&out, t)
	println(fmt.Sprintf("Copy into Buffer %d %s len=%d reads=%d", n, errText(err), out.Len(), t.reads))

	c := &capped{max: 3}
	n, err = io.Copy(c, src("abcdefgh", 8))
	println(fmt.Sprintf("Copy short write %d %q short=%t", n, c.buf.String(), err == io.ErrShortWrite))
	n, err = io.Copy(&capped{max: 100}, &trickle{data: []byte("abc"), step: 2, fail: errDisk})
	println(fmt.Sprintf("Copy read error %d %s", n, errText(err)))

	out.Reset()
	n, err = io.CopyN(&out, strings.NewReader("copy exactly this"), 10)
	println(fmt.Sprintf("CopyN %d %q %s", n, out.String(), errText(err)))
	n, err = io.CopyN(&out, src("tiny", 2), 10)
	println(fmt.Sprintf("CopyN short %d eof=%t", n, err == io.EOF))

	c = &capped{max: 100}
	n, err = io.CopyBuffer(c, src("buffered copy", 13), make([]byte, 4))
	println(fmt.Sprintf("CopyBuffer %d %q %s", n, c.buf.String(), errText(err)))
	println("CopyBuffer empty panics", panics(func() { io.CopyBuffer(c, src("x", 1), []byte{}) }))

	n, err = io.Copy(io.Discard, src(strings.Repeat("d", 20000), 4096))
	println(fmt.Sprintf("Discard %d %s", n, errText(err)))

	var a, b bytes.Buffer
	w := io.MultiWriter(&a, io.MultiWriter(&b, io.Discard))
	k, err := io.WriteString(w, "fan out")
	println(fmt.Sprintf("MultiWriter %d %q %q %s", k, a.String(), b.String(), errText(err)))
	k, err = io.MultiWriter(&a, broken{}, &b).Write([]byte("!"))
	println(fmt.Sprintf("MultiWriter error %d %s %q %q", k, errText(err), a.String(), b.String()))

	rc := io.NopCloser(strings.NewReader("nop"))
	_, keepsWriterTo := rc.(io.WriterTo)
	all, _ := io.ReadAll(rc)
	println(fmt.Sprintf("NopCloser %q close=%s writerTo=%t", all, errText(rc.Close()), keepsWriterTo))
	_, plainWriterTo := io.NopCloser(src("x", 1)).(io.WriterTo)
	println("NopCloser plain writerTo", plainWriterTo)
}

func panics(f func()) (p bool) {
	defer func() { p = recover() != nil }()
	f()
	return false
}

func sections() {
	base := strings.NewReader("0123456789abcdefghij")
	s := io.NewSectionReader(base, 5, 10)
	buf := make([]byte, 4)
	n, err := s.Read(buf)
	println(fmt.Sprintf("Section Read %d %q %s size=%d", n, buf[:n], errText(err), s.Size()))
	pos, err := s.Seek(-3, io.SeekEnd)
	println(fmt.Sprintf("Section Seek end %d %s", pos, errText(err)))
	all, _ := io.ReadAll(s)
	println(fmt.Sprintf("Section rest %q", all))
	n, err = s.ReadAt(buf, 8)
	println(fmt.Sprintf("Section ReadAt tail %d %q eof=%t", n, buf[:n], err == io.EOF))
	n, err = s.ReadAt(buf, 20)
	println(fmt.Sprintf("Section ReadAt past %d eof=%t", n, err == io.EOF))
	_, err = s.Seek(-1, io.SeekStart)
	println("Section Seek negative", errText(err))
	_, err = s.Seek(0, 7)
	println("Section Seek whence", errText(err))
	r, off, size := s.Outer()
	println(fmt.Sprintf("Section Outer %t %d %d", r == io.ReaderAt(base), off, size))

	sink := &sparse{}
	o := io.NewOffsetWriter(sink, 2)
	io.WriteString(o, "abc")
	o.Seek(5, io.SeekStart)
	o.Write([]byte("Z"))
	o.WriteAt([]byte("q"), 0)
	_, err = o.Seek(0, io.SeekEnd)
	println(fmt.Sprintf("OffsetWriter %q %s", sink.data, errText(err)))
}

func byteReaders() {
	r := bytes.NewReader([]byte("héllo, wörld"))
	ch, size, _ := r.ReadRune()
	ch2, size2, _ := r.ReadRune()
	println(fmt.Sprintf("bytes ReadRune %q %d %q %d len=%d", ch, size, ch2, size2, r.Len()))
	println("bytes UnreadRune", errText(r.UnreadRune()), errText(r.UnreadRune()), r.Len())
	pos, _ := r.Seek(-6, io.SeekEnd)
	b, _ := r.ReadByte()
	println(fmt.Sprintf("bytes Seek %d %q unread=%s", pos, b, errText(r.UnreadByte())))
	buf := make([]byte, 8)
	n, err := r.ReadAt(buf, 3)
	println(fmt.Sprintf("bytes ReadAt %d %q %s", n, buf[:n], errText(err)))
	n, err = r.ReadAt(buf, 9)
	println(fmt.Sprintf("bytes ReadAt tail %d %q eof=%t", n, buf[:n], err == io.EOF))
	_, err = r.ReadAt(buf, -1)
	println("bytes ReadAt negative", errText(err))
	var out bytes.Buffer
	m, err := r.WriteTo(&out)
	println(fmt.Sprintf("bytes WriteTo %d %q %s size=%d", m, out.String(), errText(err), r.Size()))
	n, err = r.Read(buf)
	println(fmt.Sprintf("bytes drained %d eof=%t", n, err == io.EOF))
	r.Reset([]byte("again"))
	pos, err = r.Seek(100, io.SeekCurrent)
	n, err2 := r.Read(buf)
	println(fmt.Sprintf("bytes past end %d %s %d eof=%t", pos, errText(err), n, err2 == io.EOF))
	_, err = r.Seek(-200, io.SeekCurrent)
	println("bytes Seek negative", errText(err))

	s := strings.NewReader("ünïcode")
	ch, size, _ = s.ReadRune()
	println(fmt.Sprintf("strings ReadRune %q %d unread=%s len=%d", ch, size, errText(s.UnreadRune()), s.Len()))
	s.Seek(2, io.SeekStart)
	var sb strings.Builder
	m, err = s.WriteTo(&sb)
	println(fmt.Sprintf("strings WriteTo %d %q %s", m, sb.String(), errText(err)))
	println("strings UnreadRune after WriteTo", errText(s.UnreadRune()))
	var zero strings.Reader
	n, err = zero.Read(buf)
	println(fmt.Sprintf("strings zero %d eof=%t", n, err == io.EOF))

	var bb bytes.Buffer
	k, err := bb.ReadFrom(src(strings.Repeat("r", 3000), 700))
	println(fmt.Sprintf("Buffer ReadFrom %d %s len=%d", k, errText(err), bb.Len()))
	k, err = bb.ReadFrom(&trickle{data: []byte("xy"), step: 1, fail: errDisk})
	println(fmt.Sprintf("Buffer ReadFrom error %d %s len=%d", k, errText(err), bb.Len()))
	n, _ = bb.Read(buf[:3])
	k, err = bb.WriteTo(&capped{max: 10})
	println(fmt.Sprintf("Buffer WriteTo short %d %d short=%t left=%d", n, k, err == io.ErrShortWrite, bb.Len()))
	k, err = bb.WriteTo(io.Discard)
	n, err2 = bb.Read(buf)
	_, err3 := bb.ReadByte()
	println(fmt.Sprintf("Buffer drained %d %s %d eof=%t byteEOF=%t", k, errText(err), n, err2 == io.EOF, err3 == io.EOF))
	n, err = bb.Read(nil)
	println(fmt.Sprintf("Buffer empty read %d %s", n, errText(err)))
}

// stream round-trips data through interface values only.
func stream(rs io.ReadSeeker, w io.Writer) {
	rs.Seek(4, io.SeekStart)
	var buf [3]byte
	io.ReadFull(rs, buf[:])
	rs.Seek(0, io.SeekStart)
	n, err := io.Copy(w, rs)
	println(fmt.Sprintf("interfaces %q %d %s", buf[:], n, errText(err)))
}

func main() {
	readers()
	copies()
	sections()
	byteReaders()
	var out bytes.Buffer
	stream(strings.NewReader("seek and copy"), &out)
	stream(bytes.NewReader(out.Bytes()), io.Discard)
	println("stream", out.String())
}
