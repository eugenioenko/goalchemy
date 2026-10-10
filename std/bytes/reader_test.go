package bytes_test

import (
	stdbytes "bytes"
	stdio "io"
	"math/rand"
	"testing"

	"github.com/eugenioenko/goalchemy/std/bytes"
	"github.com/eugenioenko/goalchemy/std/io"
)

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestReader(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	for _, data := range samples() {
		g, w := bytes.NewReader(data), stdbytes.NewReader(data)
		for i := 0; i < 60; i++ {
			switch op := rng.Intn(9); op {
			case 0:
				size := rng.Intn(8)
				gb, wb := make([]byte, size), make([]byte, size)
				gn, ge := g.Read(gb)
				wn, we := w.Read(wb)
				eq(t, "Read", []any{gn, errText(ge), gb}, []any{wn, errText(we), wb})
			case 1:
				size, off := rng.Intn(8), int64(rng.Intn(len(data)+4))-1
				gb, wb := make([]byte, size), make([]byte, size)
				gn, ge := g.ReadAt(gb, off)
				wn, we := w.ReadAt(wb, off)
				eq(t, "ReadAt", []any{gn, errText(ge), gb}, []any{wn, errText(we), wb})
			case 2:
				gc, ge := g.ReadByte()
				wc, we := w.ReadByte()
				eq(t, "ReadByte", []any{gc, errText(ge)}, []any{wc, errText(we)})
			case 3:
				eq(t, "UnreadByte", errText(g.UnreadByte()), errText(w.UnreadByte()))
			case 4:
				gr, gs, ge := g.ReadRune()
				wr, ws, we := w.ReadRune()
				eq(t, "ReadRune", []any{gr, gs, errText(ge)}, []any{wr, ws, errText(we)})
			case 5:
				eq(t, "UnreadRune", errText(g.UnreadRune()), errText(w.UnreadRune()))
			case 6:
				off, whence := int64(rng.Intn(len(data)+8))-4, rng.Intn(4)
				gp, ge := g.Seek(off, whence)
				wp, we := w.Seek(off, whence)
				eq(t, "Seek", []any{gp, errText(ge)}, []any{wp, errText(we)})
			case 7:
				var gb, wb stdbytes.Buffer
				gn, ge := g.WriteTo(&gb)
				wn, we := w.WriteTo(&wb)
				eq(t, "WriteTo", []any{gn, errText(ge), gb.String()}, []any{wn, errText(we), wb.String()})
			case 8:
				if rng.Intn(4) == 0 {
					g.Reset(data)
					w.Reset(data)
				}
			}
			eq(t, "Len/Size", []any{g.Len(), g.Size()}, []any{w.Len(), w.Size()})
		}
	}
	var zero bytes.Reader
	n, err := zero.Read(make([]byte, 1))
	eq(t, "zero Reader", []any{n, err}, []any{0, io.EOF})
}

func TestBufferIO(t *testing.T) {
	rng := rand.New(rand.NewSource(6))
	for _, data := range samples() {
		var g bytes.Buffer
		var w stdbytes.Buffer
		gn, ge := g.ReadFrom(bytes.NewReader(data))
		wn, we := w.ReadFrom(stdbytes.NewReader(data))
		eq(t, "ReadFrom", []any{gn, ge, g.String()}, []any{wn, we, w.String()})
		for {
			size := rng.Intn(5)
			gb, wb := make([]byte, size), make([]byte, size)
			gn, ge := g.Read(gb)
			wn, we := w.Read(wb)
			eq(t, "Read", []any{gn, errText(ge), gb}, []any{wn, errText(we), wb})
			if we != nil || rng.Intn(3) == 0 {
				break
			}
		}
		var gout, wout stdbytes.Buffer
		gm, ge := g.WriteTo(&gout)
		wm, we := w.WriteTo(&wout)
		eq(t, "WriteTo", []any{gm, ge, gout.String(), g.Len()}, []any{wm, we, wout.String(), w.Len()})
	}
	var b bytes.Buffer
	_, err := b.ReadByte()
	eq(t, "ReadByte EOF", err, io.EOF)
	n, err := io.Copy(&b, bytes.NewReader([]byte("copied")))
	eq(t, "Copy into Buffer", []any{n, err, b.String()}, []any{int64(6), error(nil), "copied"})
	var _ io.ReadWriter = &b
	var _ stdio.Reader = &b
}
