package jsonvalue_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"math/rand"
	stdstrconv "strconv"
	"strings"
	"testing"

	"github.com/eugenioenko/goalchemy/std/encoding/jsonvalue"
)

type frame struct {
	object bool
	n      int
	seen   map[string]bool
}

// reference re-encodes data compactly with encoding/json's escaping, keeping
// number text and member order. It reports false for input encoding/json
// rejects and for objects with duplicate members, which jsonvalue rejects.
func reference(data []byte) (string, bool) {
	if !json.Valid(data) {
		return "", false
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var out []byte
	var stack []*frame
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return string(out), true
		}
		if err != nil {
			panic(err)
		}
		if d, ok := tok.(json.Delim); ok && (d == '}' || d == ']') {
			out = append(out, byte(d))
			stack = stack[:len(stack)-1]
			continue
		}
		if len(stack) > 0 {
			top := stack[len(stack)-1]
			if top.object && top.n%2 == 0 {
				if top.n > 0 {
					out = append(out, ',')
				}
				k := tok.(string)
				if top.seen[k] {
					return "", false
				}
				top.seen[k] = true
				b, _ := json.Marshal(k)
				out = append(append(out, b...), ':')
				top.n++
				continue
			}
			if !top.object && top.n > 0 {
				out = append(out, ',')
			}
			top.n++
		}
		switch t := tok.(type) {
		case json.Delim:
			out = append(out, byte(t))
			stack = append(stack, &frame{object: t == '{', seen: map[string]bool{}})
		case json.Number:
			out = append(out, t...)
		default:
			b, _ := json.Marshal(t)
			out = append(out, b...)
		}
	}
}

var corpus = []string{
	``, ` `, `null`, `true`, `false`, `nul`, `nulls`, `tru`, `True`, `NULL`, `0`, `-0`, `00`, `01`, `-01`, `1.`, `.1`, `1.e5`,
	`1e`, `1e+`, `1E-7`, `-`, `+1`, `1.5e+308`, `1e400`, `-1e-400`, `123456789012345678901234567890`, `0.0000000000000000001`,
	`9007199254740993`, `-9223372036854775809`, `18446744073709551616`, `""`, `"`, `"\"`, `"\\"`, `"a\/b"`, `"\b\f\n\r\t"`,
	`"Aé中"`, `"𝄞"`, `"\uD834"`, `"\uDD1E"`, `"\uD834A"`, `"\uD834𝄞"`, `"􏿿"`,
	`"\u12"`, `"\u12G4"`, `"\U0041"`, `"\x41"`, `"\'"`, "\"\x01\"", "\"\x1f\"", "\"\x7f\"", "\"\t\"", "\"\xff\"", "\"\xc3\x28\"",
	"\"\xe2\x82\"", "\"\xed\xa0\x80\"", "\"\xf4\x90\x80\x80\"", "\"\xc0\xaf\"", "\"ok\xe2\x82\xacok\"", `"<a href='x'>&amp;</a>"`,
	"\"  \"", `[]`, `{}`, `[ ]`, `{ }`, `[1,]`, `[,1]`, `[1 2]`, `[1,,2]`, `{"a"}`, `{"a":}`, `{"a" 1}`, `{"a":1,}`, `{,}`,
	`{1:2}`, `{'a':1}`, `{"a":1 "b":2}`, `{"a":1}x`, `[1]]`, `[[[]]]`, `[{"a":[{"b":null}]}]`, `{"a":1,"a":2}`, `{"a":{"b":1,"b":2}}`,
	`{"a":1,"b":{"a":2}}`, `{"":0}`, `{"a":1,"a":2}`, " \t\r\n[ 1 , 2 ]\n ", "\v1", "\xef\xbb\xbf1", "[1]\x00", `"a"  "b"`,
	`[true,false,null,"x",1.25,-3e-2,{"k":[]}]`, `{"z":1,"y":2,"x":3}`, `[-0.0e-0]`, `{"a":"\u0000"}`,
}

func check(t *testing.T, in string) {
	t.Helper()
	want, ok := reference([]byte(in))
	v, err := jsonvalue.Parse([]byte(in))
	if !ok {
		if err == nil {
			t.Fatalf("Parse(%q) accepted invalid input", in)
		}
		return
	}
	if err != nil {
		t.Fatalf("Parse(%q): %v", in, err)
	}
	got, err := jsonvalue.Encode(v)
	if err != nil || string(got) != want {
		t.Fatalf("Encode(Parse(%q)) = %s, %v want %s", in, got, err, want)
	}
}

func TestCorpusMatchesEncodingJSON(t *testing.T) {
	for _, in := range corpus {
		check(t, in)
	}
}

func randomString(r *rand.Rand) string {
	pieces := []string{"a", "Z", " ", "\"", "\\", "/", "<", ">", "&", "é", "中", "𝄞", " ", " ", "\x7f", "\xff", "\xc3",
		"\xed\xa0\x80", `\n`, `A`, `𝄞`, `\uD834`, `\uDD1E`, ` `, `\b`, `\/`, `\"`, `\\`}
	var b strings.Builder
	for n := r.Intn(8); n > 0; n-- {
		p := pieces[r.Intn(len(pieces))]
		if p == "\"" || p == "\\" {
			p = "\\" + p
		}
		b.WriteString(p)
	}
	return `"` + b.String() + `"`
}

func randomNumber(r *rand.Rand) string {
	forms := []string{"0", "-0", "7", "-12", "3.25", "1e5", "-2.5E-3", "123456789012345678901234567890", "1e400", "0.1"}
	return forms[r.Intn(len(forms))]
}

func randomValue(r *rand.Rand, depth int) string {
	ws := []string{"", " ", "\n", "\t", "\r\n "}
	w := func() string { return ws[r.Intn(len(ws))] }
	switch k := r.Intn(8); {
	case depth > 5 || k < 2:
		return randomNumber(r)
	case k < 4:
		return randomString(r)
	case k == 4:
		return []string{"null", "true", "false"}[r.Intn(3)]
	case k == 5:
		var parts []string
		for n := r.Intn(4); n > 0; n-- {
			parts = append(parts, w()+randomValue(r, depth+1)+w())
		}
		return "[" + strings.Join(parts, ",") + "]"
	default:
		var parts []string
		keys := []string{`"a"`, `"b"`, `"c"`, `"a"`, `""`}
		for n := r.Intn(4); n > 0; n-- {
			parts = append(parts, w()+keys[r.Intn(len(keys))]+w()+":"+w()+randomValue(r, depth+1))
		}
		return "{" + strings.Join(parts, ",") + "}"
	}
}

func TestRandomDocumentsMatchEncodingJSON(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 20000; i++ {
		in := randomValue(r, 0)
		check(t, in)
		if len(in) > 0 {
			b := []byte(in)
			switch r.Intn(3) {
			case 0:
				b = b[:r.Intn(len(b))]
			case 1:
				const alphabet = "{}[],:\"\\ 0a-"
				b[r.Intn(len(b))] = alphabet[r.Intn(len(alphabet))]
			default:
				b = append(b[:r.Intn(len(b))], b[r.Intn(len(b)):]...)
			}
			check(t, string(b))
		}
	}
}

func TestStringEncodingMatchesMarshal(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	inputs := []string{"", "plain", "<>&", "  ", "\x00\x01\x1f\x7f", "\xff\xfe", "\xed\xa0\x80", "é中𝄞", "\b\f\n\r\t\"\\/"}
	for i := 0; i < 5000; i++ {
		b := make([]byte, r.Intn(12))
		for j := range b {
			b[j] = byte(r.Intn(256))
		}
		inputs = append(inputs, string(b))
	}
	for _, s := range inputs {
		want, _ := json.Marshal(s)
		got, err := jsonvalue.Encode(jsonvalue.String(s))
		if err != nil || string(got) != string(want) {
			t.Fatalf("Encode(String(%q)) = %s, %v want %s", s, got, err, want)
		}
	}
}

func TestFloatMatchesMarshal(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	inputs := []float64{0, math.Copysign(0, -1), 1, -1, 0.1, 1e-6, 9.99e-7, 1e20, 1e21, 123456789, 5e-324, math.MaxFloat64, -math.MaxFloat64}
	for i := 0; i < 5000; i++ {
		inputs = append(inputs, math.Float64frombits(r.Uint64()), r.NormFloat64()*math.Pow(10, float64(r.Intn(60)-30)))
	}
	for _, f := range inputs {
		want, werr := json.Marshal(f)
		got, err := jsonvalue.Encode(jsonvalue.Float(f))
		if (werr == nil) != (err == nil) || werr == nil && string(got) != string(want) {
			t.Fatalf("Encode(Float(%v)) = %s, %v want %s, %v", f, got, err, want, werr)
		}
	}
}

func TestAccessors(t *testing.T) {
	v, err := jsonvalue.Parse([]byte(`{"s":"x","n":-42,"big":18446744073709551615,"f":1.5e3,"t":true,"z":null,"a":[1,"2",[3]],"o":{"k":"v"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if v.Kind() != jsonvalue.KindObject || v.Len() != 8 || strings.Join(v.Keys(), ",") != "s,n,big,f,t,z,a,o" {
		t.Fatalf("object shape: %v %d %v", v.Kind(), v.Len(), v.Keys())
	}
	if v.Get("s").String() != "x" || v.Get("n").Int64() != -42 || v.Get("n").Uint64() != 0 || v.Get("big").Uint64() != math.MaxUint64 {
		t.Fatal("scalar accessors")
	}
	if _, ok := v.Get("big").AsInt64(); ok {
		t.Fatal("AsInt64 accepted an out-of-range integer")
	}
	if _, ok := v.Get("f").AsInt64(); ok || v.Get("f").Float64() != 1500 || v.Get("f").Number() != "1.5e3" {
		t.Fatal("float accessors")
	}
	if !v.Get("t").Bool() || !v.Get("z").IsNull() || v.Get("z").String() != "null" || v.Get("missing").Exists() {
		t.Fatal("bool/null/missing")
	}
	if v.Get("a").Index(1).String() != "2" || v.Get("a").Index(2).Index(0).Int64() != 3 || v.Get("a").Index(9).Exists() {
		t.Fatal("array access")
	}
	if v.Get("a").String() != `[1,"2",[3]]` || v.Get("o").String() != `{"k":"v"}` || !v.Has("o") || v.Get("s").Has("x") {
		t.Fatal("String of containers")
	}
	if _, ok := v.Get("n").AsString(); ok || v.Get("missing").Get("x").Index(0).Int64() != 0 || v.Get("s").Len() != 0 {
		t.Fatal("mismatched kinds")
	}
	for _, s := range []string{"0", "-0", "1", "9223372036854775807", "-9223372036854775808", "9223372036854775808", "1.0", "1e3", "+1", "01", "", "-"} {
		gotI, okI := jsonvalue.Number(s).AsInt64()
		gotU, okU := jsonvalue.Number(s).AsUint64()
		gotF, okF := jsonvalue.Number(s).AsFloat64()
		valid := json.Valid([]byte(s))
		wantI, errI := stdstrconv.ParseInt(s, 10, 64)
		wantU, errU := stdstrconv.ParseUint(s, 10, 64)
		wantF, errF := stdstrconv.ParseFloat(s, 64)
		integral := !strings.ContainsAny(s, ".eE")
		if okI != (valid && integral && errI == nil) || okI && gotI != wantI {
			t.Fatalf("AsInt64(%q) = %d, %v", s, gotI, okI)
		}
		if okU != (valid && integral && errU == nil) || okU && gotU != wantU {
			t.Fatalf("AsUint64(%q) = %d, %v", s, gotU, okU)
		}
		if okF != (valid && errF == nil) || okF && gotF != wantF {
			t.Fatalf("AsFloat64(%q) = %v, %v", s, gotF, okF)
		}
	}
}

func TestBuild(t *testing.T) {
	v := jsonvalue.Object(
		"sub", jsonvalue.String("alice"),
		"exp", jsonvalue.Int(1735689600),
		"roles", jsonvalue.Array(jsonvalue.String("admin"), jsonvalue.Null()),
		"ratio", jsonvalue.Float(0.25),
		"big", jsonvalue.Uint(math.MaxUint64),
		"exact", jsonvalue.Number("1.10"),
		"ok", jsonvalue.Bool(true),
	)
	v = v.Set("exp", jsonvalue.Int(1)).Set("new", jsonvalue.Object())
	roles := v.Get("roles").Append(jsonvalue.String("x"))
	v = v.Set("roles", roles)
	got, err := jsonvalue.Encode(v)
	want := `{"sub":"alice","exp":1,"roles":["admin",null,"x"],"ratio":0.25,"big":18446744073709551615,"exact":1.10,"ok":true,"new":{}}`
	if err != nil || string(got) != want {
		t.Fatalf("Encode = %s, %v", got, err)
	}
	var zero jsonvalue.Value
	if b, _ := jsonvalue.Encode(zero.Set("a", jsonvalue.Array().Append(jsonvalue.Int(1)))); string(b) != `{"a":[1]}` {
		t.Fatalf("Set on zero = %s", b)
	}
}

func TestEncodeErrors(t *testing.T) {
	var invalid *jsonvalue.InvalidValueError
	for _, v := range []jsonvalue.Value{
		{}, jsonvalue.Number("01"), jsonvalue.Number(""), jsonvalue.Float(math.NaN()), jsonvalue.Float(math.Inf(1)),
		jsonvalue.Object("a", jsonvalue.Int(1), "a", jsonvalue.Int(2)), jsonvalue.Array(jsonvalue.Value{}),
	} {
		if _, err := jsonvalue.Encode(v); !errors.As(err, &invalid) {
			t.Errorf("Encode(%#v) error = %v", v, err)
		}
	}
	if _, err := jsonvalue.Encode(jsonvalue.Float(math.NaN())); err == nil || err.Error() != "jsonvalue: unsupported value: NaN" {
		t.Errorf("NaN error = %v", err)
	}
}

func TestLimits(t *testing.T) {
	var limit *jsonvalue.LimitError
	var syntax *jsonvalue.SyntaxError
	deep := func(n int) []byte { return []byte(strings.Repeat("[", n) + strings.Repeat("]", n)) }
	if _, err := jsonvalue.Parse(deep(128)); err != nil {
		t.Fatalf("depth 128: %v", err)
	}
	if _, err := jsonvalue.Parse(deep(129)); !errors.As(err, &limit) || limit.Limit != "depth" {
		t.Fatalf("depth 129: %v", err)
	}
	l := jsonvalue.DefaultLimits()
	l.Nodes = 3
	if _, err := jsonvalue.ParseWithLimits([]byte(`[1,2,3]`), l); !errors.As(err, &limit) || limit.Limit != "nodes" {
		t.Fatalf("nodes: %v", err)
	}
	l = jsonvalue.DefaultLimits()
	l.StringBytes = 3
	if _, err := jsonvalue.ParseWithLimits([]byte(`"abc"`), l); err != nil {
		t.Fatalf("string at limit: %v", err)
	}
	if _, err := jsonvalue.ParseWithLimits([]byte(`"abcd"`), l); !errors.As(err, &limit) || limit.Limit != "string bytes" {
		t.Fatalf("string bytes: %v", err)
	}
	l = jsonvalue.DefaultLimits()
	l.Bytes = 4
	if _, err := jsonvalue.ParseWithLimits([]byte(`[1,2]`), l); !errors.As(err, &limit) || limit.Limit != "bytes" {
		t.Fatalf("bytes: %v", err)
	}
	if _, err := jsonvalue.EncodeWithLimits(jsonvalue.String("hello"), l); !errors.As(err, &limit) {
		t.Fatalf("encode bytes: %v", err)
	}
	l = jsonvalue.DefaultLimits()
	l.Depth = jsonvalue.MaxDepth + 1
	if _, err := jsonvalue.ParseWithLimits([]byte(`1`), l); !errors.As(err, &limit) || limit.Limit != "limits" {
		t.Fatalf("invalid limits: %v", err)
	}
	if _, err := jsonvalue.Parse([]byte(`{"a":1,"a":2}`)); !errors.As(err, &syntax) || syntax.Offset != 7 {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := jsonvalue.Parse([]byte(`[1,x]`)); err == nil || err.Error() != `jsonvalue: invalid character 'x' looking for beginning of value at offset 3` {
		t.Fatalf("syntax message: %v", err)
	}
}
