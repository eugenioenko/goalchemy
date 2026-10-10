package io_test

import (
	stdbytes "bytes"
	"errors"
	stdio "io"
	"math/rand"
	"reflect"
	"testing"

	"github.com/eugenioenko/goalchemy/std/io"
)

func msg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func eq(t *testing.T, name string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: got %#v want %#v", name, got, want)
	}
}

// chunks reads at most step bytes per call and fails with fail, if set,
// instead of EOF.
type chunks struct {
	data []byte
	step int
	fail error
	eof  error
}

func (c *chunks) Read(p []byte) (int, error) {
	if len(c.data) == 0 {
		if c.fail != nil {
			return 0, c.fail
		}
		return 0, c.eof
	}
	n := c.step
	if n > len(p) {
		n = len(p)
	}
	if n > len(c.data) {
		n = len(c.data)
	}
	copy(p, c.data[:n])
	c.data = c.data[n:]
	return n, nil
}

func pair(data []byte, step int, fail string) (*chunks, *chunks) {
	var g, w error
	if fail != "" {
		g, w = errors.New(fail), errors.New(fail)
	}
	return &chunks{data, step, g, io.EOF}, &chunks{data, step, w, stdio.EOF}
}

type limited struct {
	buf stdbytes.Buffer
	max int
}

func (l *limited) Write(p []byte) (int, error) {
	if len(p) > l.max {
		l.buf.Write(p[:l.max])
		return l.max, nil
	}
	return l.buf.Write(p)
}

func cases() [][]byte {
	rng := rand.New(rand.NewSource(7))
	out := [][]byte{nil, {}, []byte("x")}
	for i := 0; i < 60; i++ {
		b := make([]byte, rng.Intn(70000))
		rng.Read(b)
		out = append(out, b)
	}
	return out
}

func TestReadHelpers(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for _, data := range cases() {
		for _, fail := range []string{"", "boom"} {
			step := 1 + rng.Intn(5000)
			size := rng.Intn(len(data) + 10)
			min := rng.Intn(size + 2)

			g, w := pair(data, step, fail)
			gb, wb := make([]byte, size), make([]byte, size)
			gn, ge := io.ReadAtLeast(g, gb, min)
			wn, we := stdio.ReadAtLeast(w, wb, min)
			eq(t, "ReadAtLeast", []any{gn, msg(ge), gb}, []any{wn, msg(we), wb})

			g, w = pair(data, step, fail)
			gn, ge = io.ReadFull(g, gb)
			wn, we = stdio.ReadFull(w, wb)
			eq(t, "ReadFull", []any{gn, msg(ge), gb}, []any{wn, msg(we), wb})

			g, w = pair(data, step, fail)
			ga, ge := io.ReadAll(g)
			wa, we := stdio.ReadAll(w)
			eq(t, "ReadAll", []any{string(ga), msg(ge)}, []any{string(wa), msg(we)})

			n := int64(rng.Intn(len(data) + 10))
			g, w = pair(data, step, fail)
			ga, ge = io.ReadAll(io.LimitReader(g, n))
			wa, we = stdio.ReadAll(stdio.LimitReader(w, n))
			eq(t, "LimitReader", []any{string(ga), msg(ge)}, []any{string(wa), msg(we)})
		}
	}
	if _, err := io.ReadFull(&chunks{eof: io.EOF}, make([]byte, 3)); err != io.EOF {
		t.Fatalf("ReadFull on empty input: %v", err)
	}
	if _, err := io.ReadAtLeast(&chunks{eof: io.EOF}, make([]byte, 1), 2); err != io.ErrShortBuffer {
		t.Fatalf("ReadAtLeast short buffer: %v", err)
	}
}

func TestCopy(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	for _, data := range cases() {
		for _, fail := range []string{"", "boom"} {
			step := 1 + rng.Intn(5000)
			g, w := pair(data, step, fail)
			var gb, wb stdbytes.Buffer
			gn, ge := io.Copy(&gb, g)
			wn, we := stdio.Copy(&wb, w)
			eq(t, "Copy", []any{gn, msg(ge), gb.String()}, []any{wn, msg(we), wb.String()})

			n := int64(rng.Intn(len(data) + 10))
			g, w = pair(data, step, fail)
			gb.Reset()
			wb.Reset()
			gn, ge = io.CopyN(&gb, g, n)
			wn, we = stdio.CopyN(&wb, w, n)
			eq(t, "CopyN", []any{gn, msg(ge), gb.String()}, []any{wn, msg(we), wb.String()})

			g, w = pair(data, step, fail)
			gb.Reset()
			wb.Reset()
			buf := make([]byte, 1+rng.Intn(100))
			gn, ge = io.CopyBuffer(&gb, g, buf)
			wn, we = stdio.CopyBuffer(&wb, w, buf)
			eq(t, "CopyBuffer", []any{gn, msg(ge), gb.String()}, []any{wn, msg(we), wb.String()})

			max := 1 + rng.Intn(4000)
			g, w = pair(data, step, fail)
			gl, wl := &limited{max: max}, &limited{max: max}
			gn, ge = io.Copy(gl, g)
			wn, we = stdio.Copy(wl, w)
			eq(t, "short write", []any{gn, msg(ge), gl.buf.String()}, []any{wn, msg(we), wl.buf.String()})
		}
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("CopyBuffer with an empty buffer did not panic")
			}
		}()
		io.CopyBuffer(io.Discard, &chunks{eof: io.EOF}, []byte{})
	}()
}

type at struct{ data []byte }

func (a at) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(a.data)) {
		return 0, io.EOF
	}
	n := copy(p, a.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func TestSectionReader(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for _, data := range cases() {
		off := int64(rng.Intn(len(data) + 5))
		n := int64(rng.Intn(len(data) + 5))
		g := io.NewSectionReader(at{data}, off, n)
		w := stdio.NewSectionReader(stdbytes.NewReader(data), off, n)
		eq(t, "Size", g.Size(), w.Size())
		for i := 0; i < 40; i++ {
			switch rng.Intn(3) {
			case 0:
				size := rng.Intn(3000)
				gb, wb := make([]byte, size), make([]byte, size)
				gn, ge := g.Read(gb)
				wn, we := w.Read(wb)
				eq(t, "Read", []any{gn, msg(ge), gb[:gn]}, []any{wn, msg(we), wb[:wn]})
			case 1:
				size, o := rng.Intn(3000), int64(rng.Intn(int(n)+5))-2
				gb, wb := make([]byte, size), make([]byte, size)
				gn, ge := g.ReadAt(gb, o)
				wn, we := w.ReadAt(wb, o)
				eq(t, "ReadAt", []any{gn, msg(ge), gb[:gn]}, []any{wn, msg(we), wb[:wn]})
			case 2:
				o, whence := int64(rng.Intn(int(n)+20))-10, rng.Intn(4)
				gp, ge := g.Seek(o, whence)
				wp, we := w.Seek(o, whence)
				eq(t, "Seek", []any{gp, ge == nil}, []any{wp, we == nil})
			}
		}
		gr, go1, gn := g.Outer()
		_, wo, wn := w.Outer()
		eq(t, "Outer", []any{gr.(at).data, go1, gn}, []any{data, wo, wn})
	}
}

type at2 struct{ buf []byte }

func (a *at2) WriteAt(p []byte, off int64) (int, error) {
	for int64(len(a.buf)) < off+int64(len(p)) {
		a.buf = append(a.buf, 0)
	}
	return copy(a.buf[off:], p), nil
}

func TestOffsetWriter(t *testing.T) {
	g, w := &at2{}, &at2{}
	gw, ww := io.NewOffsetWriter(g, 3), stdio.NewOffsetWriter(w, 3)
	gw.Write([]byte("hello"))
	ww.Write([]byte("hello"))
	gp, ge := gw.Seek(2, io.SeekStart)
	wp, we := ww.Seek(2, stdio.SeekStart)
	eq(t, "Seek", []any{gp, ge == nil}, []any{wp, we == nil})
	gw.Write([]byte("XY"))
	ww.Write([]byte("XY"))
	gw.WriteAt([]byte("Z"), 9)
	ww.WriteAt([]byte("Z"), 9)
	_, ge = gw.Seek(-1, io.SeekStart)
	_, we = ww.Seek(-1, stdio.SeekStart)
	_, gx := gw.Seek(0, io.SeekEnd)
	_, wx := ww.Seek(0, stdio.SeekEnd)
	eq(t, "errors", []any{g.buf, ge != nil, gx != nil}, []any{w.buf, we != nil, wx != nil})
}

func TestCombinators(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	for _, data := range cases() {
		cut1 := rng.Intn(len(data) + 1)
		cut2 := cut1 + rng.Intn(len(data)-cut1+1)
		parts := [][]byte{data[:cut1], nil, data[cut1:cut2], data[cut2:]}
		var gs []io.Reader
		var ws []stdio.Reader
		for _, p := range parts {
			step := 1 + rng.Intn(3000)
			g, w := pair(p, step, "")
			gs = append(gs, g)
			ws = append(ws, w)
		}
		var g1, g2, w1, w2 stdbytes.Buffer
		gr := io.TeeReader(io.MultiReader(io.MultiReader(gs...)), io.MultiWriter(&g1, &g2))
		wr := stdio.TeeReader(stdio.MultiReader(stdio.MultiReader(ws...)), stdio.MultiWriter(&w1, &w2))
		ga, ge := io.ReadAll(gr)
		wa, we := stdio.ReadAll(wr)
		eq(t, "tee multi", []any{string(ga), msg(ge), g1.String(), g2.String()},
			[]any{string(wa), msg(we), w1.String(), w2.String()})
	}

	var a, b stdbytes.Buffer
	n, err := io.WriteString(io.MultiWriter(&a, io.MultiWriter(&b, io.Discard)), "abc")
	eq(t, "MultiWriter WriteString", []any{n, err, a.String(), b.String()}, []any{3, error(nil), "abc", "abc"})
	short := &limited{max: 1}
	n, err = io.MultiWriter(short).Write([]byte("abc"))
	eq(t, "MultiWriter short", []any{n, err}, []any{1, io.ErrShortWrite})
	_, err = io.TeeReader(&chunks{data: []byte("x"), step: 1, eof: io.EOF}, failing{}).Read(make([]byte, 4))
	eq(t, "TeeReader write error", msg(err), "nope")

	m, err := io.Copy(io.Discard, &chunks{data: make([]byte, 20000), step: 999, eof: io.EOF})
	eq(t, "Discard", []any{m, err}, []any{int64(20000), error(nil)})
	m, err = io.Copy(io.Discard, &chunks{data: []byte("ab"), step: 1, fail: errors.New("bad")})
	eq(t, "Discard error", []any{m, msg(err)}, []any{int64(2), "bad"})

	rc := io.NopCloser(&chunks{data: []byte("q"), step: 1, eof: io.EOF})
	if _, ok := rc.(io.WriterTo); ok || rc.Close() != nil {
		t.Fatal("NopCloser over a plain reader")
	}
}

type failing struct{}

func (failing) Write([]byte) (int, error) { return 0, errors.New("nope") }

type writerTo struct{ chunks }

func (w *writerTo) WriteTo(dst io.Writer) (int64, error) {
	n, err := dst.Write(w.data)
	return int64(n), err
}

type readerFrom struct{ got []byte }

func (r *readerFrom) Write(p []byte) (int, error) { return len(p), nil }

func (r *readerFrom) ReadFrom(src io.Reader) (int64, error) {
	b, err := io.ReadAll(src)
	r.got = b
	return int64(len(b)), err
}

func TestCopyInterfaces(t *testing.T) {
	var out stdbytes.Buffer
	src := &writerTo{chunks{data: []byte("via WriteTo"), step: 1}}
	n, err := io.Copy(&out, src)
	eq(t, "WriterTo", []any{n, err, out.String()}, []any{int64(11), error(nil), "via WriteTo"})

	rc := io.NopCloser(src)
	if _, ok := rc.(io.WriterTo); !ok {
		t.Fatal("NopCloser drops WriterTo")
	}

	dst := &readerFrom{}
	n, err = io.Copy(dst, &chunks{data: []byte("via ReadFrom"), step: 2, eof: io.EOF})
	eq(t, "ReaderFrom", []any{n, err, string(dst.got)}, []any{int64(12), error(nil), "via ReadFrom"})
}

func TestErrors(t *testing.T) {
	for _, c := range []struct {
		g error
		w error
	}{
		{io.EOF, stdio.EOF}, {io.ErrUnexpectedEOF, stdio.ErrUnexpectedEOF},
		{io.ErrShortWrite, stdio.ErrShortWrite}, {io.ErrShortBuffer, stdio.ErrShortBuffer},
		{io.ErrNoProgress, stdio.ErrNoProgress}, {io.ErrClosedPipe, stdio.ErrClosedPipe},
	} {
		eq(t, "message", c.g.Error(), c.w.Error())
	}
	eq(t, "whence", []int{io.SeekStart, io.SeekCurrent, io.SeekEnd},
		[]int{stdio.SeekStart, stdio.SeekCurrent, stdio.SeekEnd})
}
