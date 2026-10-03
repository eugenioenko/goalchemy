package binary_test

import (
	"bytes"
	stdbinary "encoding/binary"
	"math"
	"math/rand"
	"testing"

	"github.com/eugenioenko/goalchemy/std/encoding/binary"
)

func TestMatchesStd(t *testing.T) {
	rng := rand.New(rand.NewSource(9))
	vals := []uint64{0, 1, 127, 128, 255, 256, 16383, 16384, math.MaxUint32, math.MaxUint64, 1 << 63}
	for i := 0; i < 2000; i++ {
		vals = append(vals, rng.Uint64()>>uint(rng.Intn(64)))
	}
	for _, v := range vals {
		for _, o := range []struct {
			g binary.ByteOrder
			a binary.AppendByteOrder
			w stdbinary.ByteOrder
			x stdbinary.AppendByteOrder
		}{{binary.BigEndian, binary.BigEndian, stdbinary.BigEndian, stdbinary.BigEndian}, {binary.LittleEndian, binary.LittleEndian, stdbinary.LittleEndian, stdbinary.LittleEndian}} {
			if o.g.String() != o.w.String() {
				t.Fatal("String")
			}
			g, w := make([]byte, 8), make([]byte, 8)
			o.g.PutUint64(g, v)
			o.w.PutUint64(w, v)
			o.g.PutUint32(g[1:], uint32(v))
			o.w.PutUint32(w[1:], uint32(v))
			o.g.PutUint16(g[5:], uint16(v))
			o.w.PutUint16(w[5:], uint16(v))
			if !bytes.Equal(g, w) || o.g.Uint64(g) != o.w.Uint64(w) || o.g.Uint32(g[2:]) != o.w.Uint32(w[2:]) || o.g.Uint16(g[3:]) != o.w.Uint16(w[3:]) {
				t.Fatalf("%s %d", o.w, v)
			}
			if !bytes.Equal(o.a.AppendUint64(nil, v), o.x.AppendUint64(nil, v)) || !bytes.Equal(o.a.AppendUint32(nil, uint32(v)), o.x.AppendUint32(nil, uint32(v))) || !bytes.Equal(o.a.AppendUint16(nil, uint16(v)), o.x.AppendUint16(nil, uint16(v))) {
				t.Fatal("Append")
			}
		}
		if !bytes.Equal(binary.AppendUvarint(nil, v), stdbinary.AppendUvarint(nil, v)) || !bytes.Equal(binary.AppendVarint(nil, int64(v)), stdbinary.AppendVarint(nil, int64(v))) {
			t.Fatal("AppendVarint")
		}
		g, w := make([]byte, 10), make([]byte, 10)
		if binary.PutUvarint(g, v) != stdbinary.PutUvarint(w, v) || !bytes.Equal(g, w) || binary.PutVarint(g, int64(v)) != stdbinary.PutVarint(w, int64(v)) || !bytes.Equal(g, w) {
			t.Fatal("Put")
		}
	}
	for i := 0; i < 5000; i++ {
		b := make([]byte, rng.Intn(12))
		for j := range b {
			b[j] = byte(rng.Intn(256))
		}
		gu, gn := binary.Uvarint(b)
		wu, wn := stdbinary.Uvarint(b)
		gi, gm := binary.Varint(b)
		wi, wm := stdbinary.Varint(b)
		if gu != wu || gn != wn || gi != wi || gm != wm {
			t.Fatalf("Uvarint(%x) = %d,%d want %d,%d", b, gu, gn, wu, wn)
		}
	}
}
