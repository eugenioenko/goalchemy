package main

import (
	"github.com/eugenioenko/goalchemy/std/encoding/json"
	"github.com/eugenioenko/goalchemy/std/errors"
)

type Celsius float64

func (c Celsius) MarshalJSON() ([]byte, error) {
	return []byte(` { "c" : "<deg>" , "v": [ 1 , 2 ] } `), nil
}

type Ptr int

func (p *Ptr) MarshalJSON() ([]byte, error) { return []byte(`"ptr"`), nil }

type Color int

func (c Color) MarshalText() ([]byte, error) {
	return []byte([]string{"red", "gr<ee>n", "b\xffue"}[c]), nil
}

type PtrText int

func (p *PtrText) MarshalText() ([]byte, error) { return []byte("ptr-text"), nil }

type Both int

func (Both) MarshalJSON() ([]byte, error) { return []byte(`"json"`), nil }
func (Both) MarshalText() ([]byte, error) { return []byte("text"), nil }

type Fail int

func (Fail) MarshalJSON() ([]byte, error) { return nil, errFail }

type BadJSON int

func (BadJSON) MarshalJSON() ([]byte, error) { return []byte(`{bad`), nil }

type FailText int

func (FailText) MarshalText() ([]byte, error) { return nil, errFail }

var errFail = errors.New("boom")

type Embed struct {
	Ptr
	Name string
}

type Holder struct {
	C      Celsius
	CP     *Celsius
	P      Ptr
	PS     []Ptr
	PM     map[string]Ptr
	PA     [1]Ptr
	Col    Color
	PT     PtrText
	B      Both
	Raw    json.RawMessage
	NilRaw json.RawMessage
	Num    json.Number
	Any    any
	Marsh  json.Marshaler
	Keys   map[Color]int
	E      Embed
}

type label string

func (label label) show(b []byte, err error) {
	if err != nil {
		println(string(label), "error:", err.Error())
		return
	}
	println(string(label), string(b))
}

func main() {
	deg := Celsius(1)
	h := Holder{C: 2, CP: &deg, P: 1, PS: []Ptr{1, 2}, PM: map[string]Ptr{"k": 3}, PA: [1]Ptr{4}, Col: 1, PT: 5, B: 6,
		Raw: json.RawMessage(` [ 1 , {"a" : "<"} ] `), Num: "12.50",
		Any:  []any{Celsius(3), Color(2), nil, map[string]any{"b": json.Number("7"), "a": []byte("hi")}},
		Keys: map[Color]int{2: 1, 0: 2, 1: 3}, E: Embed{Name: "e"}}
	label("holder-value").show(json.Marshal(h))
	label("holder-pointer").show(json.Marshal(&h))
	label("indent").show(json.MarshalIndent(h, "", " "))
	label("ptr-value").show(json.Marshal(Ptr(1)))
	p := Ptr(1)
	label("ptr-pointer").show(json.Marshal(&p))
	var nilPtr *Ptr
	label("nil-ptr").show(json.Marshal(nilPtr))
	label("nil-ptr-any").show(json.Marshal([]any{nilPtr}))
	var nilCel *Celsius
	label("nil-cel-any").show(json.Marshal(map[string]any{"x": nilCel}))
	label("both").show(json.Marshal(Both(1)))
	pt := PtrText(1)
	label("text-any").show(json.Marshal([]any{Color(0), &pt}))
	label("raw-top").show(json.Marshal(json.RawMessage(`{"b" : 2}`)))
	label("raw-empty").show(json.Marshal(json.RawMessage{}))
	label("raw-bad").show(json.Marshal(json.RawMessage(`[1,`)))
	label("dynamic").show(json.Marshal(map[string]any{"n": nil, "f": float32(1.5), "i": int16(-2), "u": uint(3), "s": "<s>",
		"l": []any{true, false, 1e21, []any{}}, "m": map[string]any{}}))
	var nilAny []any
	label("nil-any-slice").show(json.Marshal(nilAny))
	var nilMap map[string]any
	label("nil-any-map").show(json.Marshal(nilMap))

	_, err := json.Marshal(Fail(1))
	var me *json.MarshalerError
	println("fail", errors.As(err, &me), errors.Is(err, errFail), err.Error())
	_, err = json.Marshal([]any{Fail(1)})
	println("fail-any", err.Error())
	_, err = json.Marshal(struct{ F *Fail }{new(Fail)})
	println("fail-ptr", err.Error())
	_, err = json.Marshal(BadJSON(1))
	var se *json.SyntaxError
	println("bad-json", errors.As(err, &me), errors.As(err, &se), err.Error())
	_, err = json.Marshal(FailText(1))
	println("fail-text", err.Error())
	_, err = json.Marshal(map[FailText]int{1: 1})
	println("fail-text-key", err.Error())
}
