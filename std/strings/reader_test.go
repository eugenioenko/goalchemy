package strings_test

import (
	stdbytes "bytes"
	"math/rand"
	stdstrings "strings"
	"testing"

	"github.com/eugenioenko/goalchemy/std/io"
	"github.com/eugenioenko/goalchemy/std/strings"
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
		g, w := strings.NewReader(data), stdstrings.NewReader(data)
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
	var zero strings.Reader
	n, err := zero.Read(make([]byte, 1))
	eq(t, "zero Reader", []any{n, err}, []any{0, io.EOF})
}
