package utf8_test

import (
	"math/rand"
	"testing"
	stdutf8 "unicode/utf8"

	"github.com/eugenioenko/goalchemy/std/unicode/utf8"
)

func TestRunesMatchStd(t *testing.T) {
	for r := rune(-2); r <= stdutf8.MaxRune+2; r++ {
		if utf8.RuneLen(r) != stdutf8.RuneLen(r) || utf8.ValidRune(r) != stdutf8.ValidRune(r) {
			t.Fatalf("RuneLen/ValidRune(%U)", r)
		}
		if g, w := string(utf8.AppendRune(nil, r)), string(stdutf8.AppendRune(nil, r)); g != w {
			t.Fatalf("AppendRune(%U)", r)
		}
		var gb, wb [4]byte
		if utf8.EncodeRune(gb[:], r) != stdutf8.EncodeRune(wb[:], r) || gb != wb {
			t.Fatalf("EncodeRune(%U)", r)
		}
	}
}

func TestBytesMatchStd(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	inputs := [][]byte{nil, {}, []byte("héllo 日本 😀"), {0xff}, {0xc3}, {0xed, 0xa0, 0x80}, {0xf4, 0x90, 0x80, 0x80}, {0xe2, 0x82}, {0xc0, 0x80}}
	for i := 0; i < 20000; i++ {
		b := make([]byte, rng.Intn(8))
		for j := range b {
			b[j] = byte(rng.Intn(256))
			if rng.Intn(3) == 0 {
				b[j] = 0x80 | byte(rng.Intn(64))
			}
		}
		inputs = append(inputs, b)
	}
	for _, b := range inputs {
		s := string(b)
		gr, gn := utf8.DecodeRune(b)
		wr, wn := stdutf8.DecodeRune(b)
		if gr != wr || gn != wn {
			t.Fatalf("DecodeRune(%x) = %U,%d want %U,%d", b, gr, gn, wr, wn)
		}
		gr, gn = utf8.DecodeRuneInString(s)
		if gr != wr || gn != wn {
			t.Fatalf("DecodeRuneInString(%x)", b)
		}
		gr, gn = utf8.DecodeLastRune(b)
		wr, wn = stdutf8.DecodeLastRune(b)
		if gr != wr || gn != wn {
			t.Fatalf("DecodeLastRune(%x) = %U,%d want %U,%d", b, gr, gn, wr, wn)
		}
		gr, gn = utf8.DecodeLastRuneInString(s)
		if gr != wr || gn != wn {
			t.Fatalf("DecodeLastRuneInString(%x)", b)
		}
		if utf8.RuneCount(b) != stdutf8.RuneCount(b) || utf8.RuneCountInString(s) != stdutf8.RuneCountInString(s) {
			t.Fatalf("RuneCount(%x)", b)
		}
		if utf8.Valid(b) != stdutf8.Valid(b) || utf8.ValidString(s) != stdutf8.ValidString(s) {
			t.Fatalf("Valid(%x)", b)
		}
		for _, c := range b {
			if utf8.RuneStart(c) != stdutf8.RuneStart(c) {
				t.Fatalf("RuneStart(%x)", c)
			}
		}
	}
}
