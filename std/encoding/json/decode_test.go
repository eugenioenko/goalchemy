package json

import (
	"encoding/base64"
	stdjson "encoding/json"
	"errors"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
)

func randomDocument(r *rand.Rand, depth int) string {
	switch k := r.Intn(9); {
	case k == 0 || depth > 3 && k > 5:
		return []string{"null", "true", "false", "0", "-1.5e3", "1e400", "12345678901234567890"}[r.Intn(7)]
	case k < 4:
		return strconv.Quote(randomString(r))
	case k < 6:
		s := "["
		for n := r.Intn(4); n > 0; n-- {
			if len(s) > 1 {
				s += ","
			}
			s += randomDocument(r, depth+1)
		}
		return s + "]"
	}
	s := "{"
	for n := r.Intn(4); n > 0; n-- {
		if len(s) > 1 {
			s += ", "
		}
		s += strconv.Quote(randomString(r)) + ":" + randomDocument(r, depth+1)
	}
	return s + "}"
}

func describeError(err error) string {
	if err == nil {
		return "<nil>"
	}
	s := err.Error()
	var se *SyntaxError
	var sse *stdjson.SyntaxError
	var ute *UnmarshalTypeError
	var sute *stdjson.UnmarshalTypeError
	switch {
	case errors.As(err, &se):
		s += " @" + strconv.FormatInt(se.Offset, 10)
	case errors.As(err, &sse):
		s += " @" + strconv.FormatInt(sse.Offset, 10)
	case errors.As(err, &ute):
		s += " |" + ute.Value + "|" + ute.Type + "|" + strconv.FormatInt(ute.Offset, 10) + "|" + ute.Struct + "|" + ute.Field
	case errors.As(err, &sute):
		s += " |" + sute.Value + "|" + sute.Type.String() + "|" + strconv.FormatInt(sute.Offset, 10) + "|" + sute.Struct + "|" + sute.Field
	}
	return s
}

func TestUnmarshalAnyMatchesEncodingJSON(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	for i := 0; i < 20000; i++ {
		doc := randomDocument(r, 0)
		if r.Intn(4) == 0 {
			doc = mutate(r, doc)
		}
		var got, want any
		gerr := unmarshalAny([]byte(doc), &got)
		werr := stdjson.Unmarshal([]byte(doc), &want)
		if describeError(gerr) != describeError(werr) || !reflect.DeepEqual(got, want) {
			t.Fatalf("Unmarshal(%q):\n got %#v %s\nwant %#v %s", doc, got, describeError(gerr), want, describeError(werr))
		}
	}
}

func TestUnmarshalAnyTargets(t *testing.T) {
	for _, doc := range []string{`{"a":[1,"x"]}`, `[1,{"b":null}]`, `"s"`, `true`, `1.5`, `null`, `{"a":1} x`, `[1`} {
		var n Number
		var raw RawMessage
		for _, target := range []struct {
			got, want any
		}{{new(map[string]any), new(map[string]any)}, {new([]any), new([]any)}, {new(string), new(string)}, {new(bool), new(bool)}, {new(float64), new(float64)}} {
			gerr := unmarshalAny([]byte(doc), target.got)
			werr := stdjson.Unmarshal([]byte(doc), target.want)
			if describeError(gerr) != describeError(werr) || !reflect.DeepEqual(reflect.ValueOf(target.got).Elem().Interface(), reflect.ValueOf(target.want).Elem().Interface()) {
				t.Errorf("Unmarshal(%q, %T): got %s, want %s", doc, target.got, describeError(gerr), describeError(werr))
			}
		}
		if err := unmarshalAny([]byte(doc), &n); err == nil && doc == "1.5" && n != "1.5" {
			t.Errorf("Number = %q", n)
		}
		if err := unmarshalAny([]byte(doc), &raw); err == nil && string(raw) != doc {
			t.Errorf("RawMessage = %q, want %q", raw, doc)
		}
	}
	if err := unmarshalAny([]byte(`1`), 5); err == nil || err.Error() != "json: Unmarshal(non-pointer int)" {
		t.Errorf("non-pointer: %v", err)
	}
	var nilMap *map[string]any
	if err := unmarshalAny([]byte(`1`), nilMap); err == nil || err.Error() != "json: Unmarshal(nil *map[string]interface {})" {
		t.Errorf("nil pointer: %v", err)
	}
}

func TestBase64MatchesEncodingBase64(t *testing.T) {
	r := rand.New(rand.NewSource(4))
	alphabet := "ABCabc019+/=\r\n!-_ "
	for i := 0; i < 50000; i++ {
		b := make([]byte, r.Intn(12))
		for j := range b {
			b[j] = alphabet[r.Intn(len(alphabet))]
		}
		got, gerr := decodeBase64(string(b))
		want, werr := base64.StdEncoding.DecodeString(string(b))
		if (gerr == nil) != (werr == nil) || gerr != nil && gerr.Error() != werr.Error() || gerr == nil && string(got) != string(want) {
			t.Fatalf("decode(%q) = %q, %v; want %q, %v", b, got, gerr, want, werr)
		}
	}
}

type quotedTarget struct {
	S string `json:",string"`
}

func TestQuotedStringMatchesEncodingJSON(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	inner := []string{`"a"`, `"a`, `a"`, ` "a"`, `"a" `, `"\x"`, `"\u12"`, `"\ud800"`, `"\ud800A"`, `"😀"`, "\"\x01\"", "\"\xff\"", `""`, `"\""`, `12`, `null`}
	for i := 0; i < 5000; i++ {
		s := inner[r.Intn(len(inner))]
		if r.Intn(2) == 0 {
			s = mutate(r, s)
		}
		doc := `{"S":` + strconv.Quote(s) + `}`
		if !stdjson.Valid([]byte(doc)) {
			continue
		}
		var want quotedTarget
		werr := stdjson.Unmarshal([]byte(doc), &want)
		gerr := quotedStringError([]byte(s))
		var sute *stdjson.UnmarshalTypeError
		wantMsg := "<nil>"
		if errors.As(werr, &sute) && errors.Unwrap(sute) != nil {
			wantMsg = errors.Unwrap(sute).Error()
		} else if werr != nil {
			wantMsg = werr.Error()
		}
		gotMsg := "<nil>"
		if gerr != nil {
			gotMsg = gerr.Error()
		}
		if s == "null" {
			continue
		}
		if gotMsg != wantMsg {
			t.Fatalf("quoted %q: got %s, want %s", s, gotMsg, wantMsg)
		}
		if gerr == nil && unquote([]byte(s)) != want.S {
			t.Fatalf("quoted %q: got %q, want %q", s, unquote([]byte(s)), want.S)
		}
	}
}
