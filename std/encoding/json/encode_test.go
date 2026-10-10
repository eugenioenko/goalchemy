package json

import (
	"bytes"
	stdjson "encoding/json"
	"math"
	"math/rand"
	"strings"
	"testing"
)

func randomString(r *rand.Rand) string {
	alphabet := []string{"a", "Z", "0", " ", "\"", "\\", "/", "<", ">", "&", "\n", "\t", "\x00", "\x1f", "\x7f",
		"\u00e9", "\u2028", "\u2029", "\U0001f600", "\xff", "\xc3", "\xed\xa0\x80", "\uFFFD"}
	var b strings.Builder
	for n := r.Intn(8); n > 0; n-- {
		b.WriteString(alphabet[r.Intn(len(alphabet))])
	}
	return b.String()
}

func randomFloat(r *rand.Rand) float64 {
	switch r.Intn(4) {
	case 0:
		return float64(r.Intn(2000) - 1000)
	case 1:
		return r.NormFloat64() * math.Pow(10, float64(r.Intn(60)-30))
	case 2:
		for {
			f := math.Float64frombits(r.Uint64())
			if !math.IsNaN(f) && !math.IsInf(f, 0) {
				return f
			}
		}
	}
	return []float64{0, math.Copysign(0, -1), 1e-6, 1e-7, 1e20, 1e21, math.MaxFloat64, math.SmallestNonzeroFloat64}[r.Intn(8)]
}

func randomValue(r *rand.Rand, depth int) any {
	k := r.Intn(14)
	if depth > 4 && k >= 12 {
		k = r.Intn(12)
	}
	switch k {
	case 0:
		return nil
	case 1:
		return r.Intn(2) == 0
	case 2:
		return r.Int63() - r.Int63()
	case 3:
		return int8(r.Intn(256) - 128)
	case 4:
		return r.Uint64()
	case 5:
		return uint16(r.Intn(65536))
	case 6:
		return randomFloat(r)
	case 7:
		return float32(randomFloat(r))
	case 8, 9:
		return randomString(r)
	case 10:
		if r.Intn(3) == 0 {
			return []byte(nil)
		}
		return []byte(randomString(r))
	case 11:
		return int(r.Int31())
	case 12:
		if r.Intn(5) == 0 {
			return []any(nil)
		}
		a := []any{}
		for n := r.Intn(4); n > 0; n-- {
			a = append(a, randomValue(r, depth+1))
		}
		return a
	}
	if r.Intn(5) == 0 {
		return map[string]any(nil)
	}
	m := map[string]any{}
	for n := r.Intn(4); n > 0; n-- {
		m[randomString(r)] = randomValue(r, depth+1)
	}
	return m
}

func TestMarshalAnyMatchesEncodingJSON(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 20000; i++ {
		v := randomValue(r, 0)
		want, werr := stdjson.Marshal(v)
		got, gerr := marshalAny(v)
		if (werr != nil) != (gerr != nil) || !bytes.Equal(got, want) {
			t.Fatalf("%#v:\n got %q %v\nwant %q %v", v, got, gerr, want, werr)
		}
		wantIndent, _ := stdjson.MarshalIndent(v, "#", "\t")
		gotIndent, _ := marshalIndentAny(v, "#", "\t")
		if !bytes.Equal(gotIndent, wantIndent) {
			t.Fatalf("%#v indent:\n got %q\nwant %q", v, gotIndent, wantIndent)
		}
	}
}

func TestUnsupportedValues(t *testing.T) {
	for _, v := range []any{math.NaN(), math.Inf(1), float32(math.Inf(-1)), []any{math.NaN()}} {
		_, werr := stdjson.Marshal(v)
		_, gerr := marshalAny(v)
		if werr == nil || gerr == nil || werr.Error() != gerr.Error() {
			t.Errorf("%v: got %v, want %v", v, gerr, werr)
		}
	}
}

var documents = []string{
	``, ` `, `null`, `true`, `false`, `0`, `-0`, `01`, `1.`, `1.5e+3`, `1E-2`, `-`, `--1`, `1e`, `"`, `"a`, `"\u12"`, `"\uD800"`,
	`"\x"`, "\"\x01\"", `[]`, `[1,]`, `[,1]`, `[1 2]`, `{}`, `{"a"}`, `{"a":}`, `{"a":1,}`, `{1:2}`, `{"a":1 "b":2}`,
	` {"a" : [ 1 , 2.5e-3 , "<&>\u2028" , {"b" : null} ] } `, `[[[[[]]]]]`, `nul`, `tru`, `fals`, `1 2`, `{} x`, "[\"\xff\"]",
	`{"a":{"b":{"c":[1,{"d":[]}]}}}`, "\t\n[\r1\n]\n", `"\/\b\f\n\r\t\\\""`, `[1e1000]`, `"\u2029"`,
}

func mutate(r *rand.Rand, s string) string {
	b := []byte(s)
	for n := r.Intn(3) + 1; n > 0; n-- {
		chars := `{}[],:"\ 0123456789.eE+-tfnul` + "\x00\xff\n"
		switch {
		case len(b) == 0 || r.Intn(3) == 0:
			i := r.Intn(len(b) + 1)
			b = append(b[:i], append([]byte{chars[r.Intn(len(chars))]}, b[i:]...)...)
		case r.Intn(2) == 0:
			i := r.Intn(len(b))
			b = append(b[:i], b[i+1:]...)
		default:
			b[r.Intn(len(b))] = chars[r.Intn(len(chars))]
		}
	}
	return string(b)
}

func TestScannerMatchesEncodingJSON(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	inputs := append([]string(nil), documents...)
	for i := 0; i < 20000; i++ {
		inputs = append(inputs, mutate(r, documents[r.Intn(len(documents))]))
	}
	for _, in := range inputs {
		src := []byte(in)
		if got, want := Valid(src), stdjson.Valid(src); got != want {
			t.Fatalf("Valid(%q) = %v, want %v", in, got, want)
		}
		var wb bytes.Buffer
		gotC, gerr := appendCompact(nil, src, false)
		werr := stdjson.Compact(&wb, src)
		if !sameError(gerr, werr) || gerr == nil && !bytes.Equal(gotC, wb.Bytes()) {
			t.Fatalf("Compact(%q):\n got %q %v\nwant %q %v", in, gotC, gerr, wb.Bytes(), werr)
		}
		wb.Reset()
		gotI, gerr := appendIndent(nil, src, ">", "  ")
		werr = stdjson.Indent(&wb, src, ">", "  ")
		if !sameError(gerr, werr) || gerr == nil && !bytes.Equal(gotI, wb.Bytes()) {
			t.Fatalf("Indent(%q):\n got %q %v\nwant %q %v", in, gotI, gerr, wb.Bytes(), werr)
		}
		wb.Reset()
		stdjson.HTMLEscape(&wb, src)
		if got := appendHTMLEscape(nil, src); !bytes.Equal(got, wb.Bytes()) {
			t.Fatalf("HTMLEscape(%q) = %q, want %q", in, got, wb.Bytes())
		}
	}
}

func sameError(got, want error) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	ws, ok := want.(*stdjson.SyntaxError)
	gs, gok := got.(*SyntaxError)
	if !ok || !gok {
		return got.Error() == want.Error()
	}
	return gs.Error() == ws.Error() && gs.Offset == ws.Offset
}

func TestNumberMarshalJSON(t *testing.T) {
	for _, n := range []Number{"", "0", "-1.5e3", "1e", "01", "abc"} {
		got, gerr := n.MarshalJSON()
		want, werr := stdjson.Marshal(stdjson.Number(n))
		if gerr != nil {
			if werr == nil {
				t.Errorf("Number(%q): unexpected error %v", n, gerr)
			}
			continue
		}
		if werr != nil || !bytes.Equal(got, want) {
			t.Errorf("Number(%q) = %q, want %q (%v)", n, got, want, werr)
		}
	}
}
