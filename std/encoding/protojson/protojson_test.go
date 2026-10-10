package protojson

import (
	"bytes"
	"encoding/json"
	"math"
	"math/rand"
	"strings"
	"testing"

	gopj "google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/eugenioenko/goalchemy/std/encoding/jsonvalue"
)

func reference(t *testing.T, m proto.Message) (string, bool) {
	t.Helper()
	b, err := gopj.Marshal(m)
	if err != nil {
		return "", false
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, b); err != nil {
		t.Fatal(err)
	}
	return buf.String(), true
}

func ours(write func(e *Encoder)) (string, bool) {
	e := &Encoder{}
	write(e)
	return string(e.buf), e.err == nil
}

func check(t *testing.T, what string, m proto.Message, write func(e *Encoder)) {
	t.Helper()
	want, wok := reference(t, m)
	got, gok := ours(write)
	if wok != gok || wok && got != want {
		t.Errorf("%s: got %q (ok %v), want %q (ok %v)", what, got, gok, want, wok)
	}
}

func TestFloatsMatchProtojson(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	values := []float64{0, math.Copysign(0, -1), 1, -1, 0.1, 1e21, 1e20, 1e-6, 1e-7, 5e-324, math.MaxFloat64,
		math.SmallestNonzeroFloat64, math.NaN(), math.Inf(1), math.Inf(-1), 123456789.125, math.MaxFloat32, 1.5e-10}
	for i := 0; i < 20000; i++ {
		values = append(values, math.Float64frombits(r.Uint64()), r.NormFloat64()*math.Pow(10, float64(r.Intn(60)-30)))
	}
	for _, f := range values {
		if math.IsNaN(f) && math.Float64bits(f) != math.Float64bits(math.NaN()) {
			continue
		}
		check(t, "double", wrapperspb.Double(f), func(e *Encoder) { e.Float64(f) })
		f32 := float32(f)
		check(t, "float", wrapperspb.Float(f32), func(e *Encoder) { e.Float32(f32) })
	}
}

func TestIntegersAndBytesMatchProtojson(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	for i := 0; i < 5000; i++ {
		n := int64(r.Uint64())
		check(t, "int64", wrapperspb.Int64(n), func(e *Encoder) { e.Int64(n) })
		check(t, "uint64", wrapperspb.UInt64(uint64(n)), func(e *Encoder) { e.Uint64(uint64(n)) })
		check(t, "int32", wrapperspb.Int32(int32(n)), func(e *Encoder) { e.Int32(int32(n)) })
		check(t, "uint32", wrapperspb.UInt32(uint32(n)), func(e *Encoder) { e.Uint32(uint32(n)) })
		b := make([]byte, r.Intn(20))
		r.Read(b)
		check(t, "bytes", wrapperspb.Bytes(b), func(e *Encoder) { e.Bytes(b) })
	}
}

func TestStringsMatchProtojson(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	samples := []string{"", "<>&", "  ", "\x7f", "\"\\/", "\xff", "a\xc0b", "😀", "퟿"}
	for c := 0; c < 0x80; c++ {
		samples = append(samples, string(rune(c)))
	}
	for i := 0; i < 2000; i++ {
		b := make([]byte, r.Intn(12))
		r.Read(b)
		samples = append(samples, string(b))
	}
	for _, s := range samples {
		check(t, "string "+strings.ToValidUTF8(s, "?"), wrapperspb.String(s), func(e *Encoder) { e.String(s) })
	}
}

func TestTimestampsAndDurationsMatchProtojson(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	for i := 0; i < 3000; i++ {
		secs := r.Int63n(maxTimestampSeconds-minTimestampSeconds+3) + minTimestampSeconds - 1
		nanos := []int32{0, 1, 999999999, 1000, 120000000, -1, 1000000000, int32(r.Int31n(1e9))}[r.Intn(8)]
		ts := &Timestamp{Seconds: secs, Nanos: nanos}
		check(t, "timestamp", &timestamppb.Timestamp{Seconds: secs, Nanos: nanos}, ts.EncodeProtoJSON)
		dsecs := r.Int63n(2*maxDurationSeconds+3) - maxDurationSeconds - 1
		dnanos := []int32{0, 1, -1, 999999999, -999999999, 1000000000, int32(r.Int31n(1e9))}[r.Intn(7)]
		d := &Duration{Seconds: dsecs, Nanos: dnanos}
		check(t, "duration", &durationpb.Duration{Seconds: dsecs, Nanos: dnanos}, d.EncodeProtoJSON)
	}
}

// decodeCase decodes input with both implementations and compares the
// re-encoded results, or that both fail.
func decodeCase(t *testing.T, input string, ref proto.Message, decode func(d *Decoder, v jsonvalue.Value) (string, bool)) {
	t.Helper()
	werr := gopj.UnmarshalOptions{DiscardUnknown: true}.Unmarshal([]byte(input), ref)
	want, _ := reference(t, ref)
	v, err := jsonvalue.Parse([]byte(input))
	got, gok := "", false
	if err == nil {
		d := &Decoder{}
		got, gok = decode(d, v)
		gok = gok && d.err == nil
	}
	if (werr == nil) != gok || gok && got != want {
		t.Errorf("decode %s: got %q (ok %v), want %q (err %v)", input, got, gok, want, werr)
	}
}

func TestLenientDecodingMatchesProtojson(t *testing.T) {
	ints := []string{`1`, `"1"`, `1.0`, `"1e2"`, `1e2`, `-0`, `"-1"`, `1.5`, `"1.5"`, `" 1"`, `"0x10"`, `2147483648`, `-2147483649`,
		`9223372036854775807`, `"9223372036854775808"`, `1e20`, `100e-2`, `120e-1`, `1E+2`, `"01"`, `true`, `null`, `[]`, `"NaN"`,
		`18446744073709551615`, `-1e0`, `0.0e5`, `1e-0`}
	for _, in := range ints {
		decodeCase(t, in, &wrapperspb.Int32Value{}, func(d *Decoder, v jsonvalue.Value) (string, bool) {
			n := d.Int32(v, "value")
			return ours(func(e *Encoder) { e.Int32(n) })
		})
		decodeCase(t, in, &wrapperspb.Int64Value{}, func(d *Decoder, v jsonvalue.Value) (string, bool) {
			n := d.Int64(v, "value")
			return ours(func(e *Encoder) { e.Int64(n) })
		})
		decodeCase(t, in, &wrapperspb.UInt64Value{}, func(d *Decoder, v jsonvalue.Value) (string, bool) {
			n := d.Uint64(v, "value")
			return ours(func(e *Encoder) { e.Uint64(n) })
		})
		decodeCase(t, in, &wrapperspb.UInt32Value{}, func(d *Decoder, v jsonvalue.Value) (string, bool) {
			n := d.Uint32(v, "value")
			return ours(func(e *Encoder) { e.Uint32(n) })
		})
		decodeCase(t, in, &wrapperspb.DoubleValue{}, func(d *Decoder, v jsonvalue.Value) (string, bool) {
			f := d.Float64(v, "value")
			return ours(func(e *Encoder) { e.Float64(f) })
		})
		decodeCase(t, in, &wrapperspb.FloatValue{}, func(d *Decoder, v jsonvalue.Value) (string, bool) {
			f := d.Float32(v, "value")
			return ours(func(e *Encoder) { e.Float32(f) })
		})
	}
	floats := []string{`"Infinity"`, `"-Infinity"`, `"NaN"`, `1e400`, `"1e400"`, `3.5e38`, `-3.5e38`, `1e-50`, `"  2"`}
	for _, in := range floats {
		decodeCase(t, in, &wrapperspb.DoubleValue{}, func(d *Decoder, v jsonvalue.Value) (string, bool) {
			f := d.Float64(v, "value")
			return ours(func(e *Encoder) { e.Float64(f) })
		})
		decodeCase(t, in, &wrapperspb.FloatValue{}, func(d *Decoder, v jsonvalue.Value) (string, bool) {
			f := d.Float32(v, "value")
			return ours(func(e *Encoder) { e.Float32(f) })
		})
	}
	b64 := []string{`""`, `"AQID"`, `"AQI="`, `"AQI"`, `"AQ=="`, `"AQ"`, `"A"`, `"-_8"`, `"+/8="`, `"-_8="`, `"AQ=I"`, `"QR=="`,
		`"AQID\n"`, `"AQ\nID"`, `"A===="`, `"!!"`, `1`}
	for _, in := range b64 {
		decodeCase(t, in, &wrapperspb.BytesValue{}, func(d *Decoder, v jsonvalue.Value) (string, bool) {
			b := d.Bytes(v, "value")
			return ours(func(e *Encoder) { e.Bytes(b) })
		})
	}
	stamps := []string{`"2024-05-06T07:08:09Z"`, `"2024-05-06T07:08:09.1+01:30"`, `"2024-05-06t07:08:09z"`, `"2024-05-06T07:08:09"`,
		`"0001-01-01T00:00:00Z"`, `"9999-12-31T23:59:59.999999999Z"`, `"2024-05-06T07:08:09.0000000001Z"`, `"2024-02-30T00:00:00Z"`, `1`}
	for _, in := range stamps {
		decodeCase(t, in, &timestamppb.Timestamp{}, func(d *Decoder, v jsonvalue.Value) (string, bool) {
			x := &Timestamp{}
			x.DecodeProtoJSON(d, v)
			return ours(x.EncodeProtoJSON)
		})
	}
	durations := []string{`"1s"`, `"1.5s"`, `"-0.5s"`, `".5s"`, `"1.s"`, `"+3s"`, `"1.0000000001s"`, `"315576000001s"`, `"s"`, `"1"`, `"01s"`, `"-s"`}
	for _, in := range durations {
		decodeCase(t, in, &durationpb.Duration{}, func(d *Decoder, v jsonvalue.Value) (string, bool) {
			x := &Duration{}
			x.DecodeProtoJSON(d, v)
			return ours(x.EncodeProtoJSON)
		})
	}
	values := []string{`null`, `{"b":2.50,"a":[1e2,"x",null,{"z":false}]}`, `[]`, `"s"`, `1e400`, `{"a":1e400}`, `-0`, `12345678901234567890`}
	for _, in := range values {
		decodeCase(t, in, &structpb.Value{}, func(d *Decoder, v jsonvalue.Value) (string, bool) {
			x := d.Value(v, "value")
			return ours(func(e *Encoder) { e.Value(x) })
		})
	}
}
