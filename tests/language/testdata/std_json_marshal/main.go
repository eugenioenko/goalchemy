package main

import (
	"github.com/eugenioenko/goalchemy/std/encoding/json"
	"github.com/eugenioenko/goalchemy/std/errors"
)

type Level int8

type Name string

type Bytes []byte

type Inner struct {
	N    int      `json:"n"`
	Tags []string `json:"tags,omitempty"`
}

type Config struct {
	Name    string         `json:"name"`
	Retries int            `json:"retries,omitempty"`
	Version int64          `json:"version,string"`
	Level   Level          `json:"level"`
	Ratio   float64        `json:"ratio"`
	Small   float32        `json:"small"`
	Data    []byte         `json:"data"`
	Raw     Bytes          `json:"raw"`
	Inner   Inner          `json:"inner"`
	Ptr     *Inner         `json:"ptr"`
	Labels  map[string]int `json:"labels"`
	Extra   any            `json:"extra"`
	Skip    string         `json:"-"`
	Dash    string         `json:"-,"`
	private int
	Plain   bool
	Punct   int `json:"a b-c.d@#$%"`
	Fixed   [2]uint16
}

type Empties struct {
	B    bool           `json:",omitempty"`
	I    int            `json:",omitempty"`
	U    uint8          `json:",omitempty"`
	F    float64        `json:",omitempty"`
	NegZ float64        `json:",omitempty"`
	S    string         `json:",omitempty"`
	Bs   []byte         `json:",omitempty"`
	Sl   []int          `json:",omitempty"`
	Esl  []int          `json:",omitempty"`
	M    map[string]int `json:",omitempty"`
	Em   map[string]int `json:",omitempty"`
	A0   [0]int         `json:",omitempty"`
	A1   [1]int         `json:",omitempty"`
	P    *int           `json:",omitempty"`
	X    any            `json:",omitempty"`
	St   struct{}       `json:",omitempty"`
	N    json.Number    `json:",omitempty"`
}

type Quoted struct {
	B  bool    `json:",string"`
	I  int     `json:",string"`
	U  uint    `json:",string"`
	F  float64 `json:",string"`
	S  string  `json:",string"`
	P  *int    `json:",string"`
	Sl []int   `json:",string"`
}

type Base struct {
	ID     int `json:"id"`
	Shared string
	Deep   string
}

type Other struct {
	Shared string
	Note   string `json:"note"`
}

type deepBase struct {
	Deep string
}

type Wrapper struct {
	Base
	*Other
	deepBase
	Own     string `json:"own"`
	Note    string
	Renamed Base `json:"renamed"`
}

type Node struct {
	Value int     `json:"value"`
	Next  *Node   `json:"next,omitempty"`
	Kids  []*Node `json:"kids,omitempty"`
}

type Numbers struct {
	N json.Number
	F []float64
	G []float32
	I []int64
	U []uint64
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
	c := Config{Name: "a<b>&c", Version: 1 << 60, Level: -3, Ratio: 1e21, Small: 0.1, Data: []byte("hi!?"),
		Raw: Bytes{0, 255}, Inner: Inner{N: 1}, Labels: map[string]int{"z": 1, "a": 2, "\u00e9": 3},
		Extra: map[string]any{"k": []any{1, "x", nil, true, 2.5, int8(-1), uint32(7), float32(0.3)}}, Dash: "dash", Punct: 9,
		Fixed: [2]uint16{1, 65535}}
	label("config").show(json.Marshal(c))
	label("pointer").show(json.Marshal(&c))
	label("indent").show(json.MarshalIndent(c.Inner, ">", "  "))
	label("indent-config").show(json.MarshalIndent(Config{}, "", "\t"))
	label("ints").show(json.Marshal([]int{1, -2, 3}))
	label("int-keys").show(json.Marshal(map[int]string{3: "c", -1: "neg", 10: "ten"}))
	label("uint-keys").show(json.Marshal(map[uint8]bool{200: true, 3: false}))
	label("name-keys").show(json.Marshal(map[Name]Name{"b": "2", "a": "1"}))
	var nilSlice []int
	label("nil-slice").show(json.Marshal(nilSlice))
	label("empty-slice").show(json.Marshal([]int{}))
	var nilMap map[string]int
	label("nil-map").show(json.Marshal(nilMap))
	label("nil").show(json.Marshal(nil))
	var nilPtr *Inner
	label("nil-ptr").show(json.Marshal(nilPtr))
	label("string").show(json.Marshal("\u2028\u2029\x01\t\n\"\\<>&\xff\xfe\u00e9\U0001f600"))
	label("named-string").show(json.Marshal(Name("x")))
	label("byte-array").show(json.Marshal([3]byte{1, 2, 3}))
	label("empty-bytes").show(json.Marshal([]byte{}))
	var nilBytes []byte
	label("nil-bytes").show(json.Marshal(nilBytes))
	for _, n := range []int{1, 2, 3, 4, 5, 6} {
		label("base64").show(json.Marshal([]byte("abcdef")[:n]))
	}
	label("empties").show(json.Marshal(Empties{NegZ: negZero(), Esl: []int{}, Em: map[string]int{}}))
	seven := 7
	label("quoted").show(json.Marshal(Quoted{B: true, I: -4, U: 5, F: 1.5, S: "s\"<", P: &seven, Sl: []int{1}}))
	label("quoted-nil").show(json.Marshal(Quoted{}))
	w := Wrapper{Base: Base{ID: 1, Shared: "base", Deep: "deep"}, Other: &Other{Shared: "other", Note: "other-note"},
		Own: "own", Note: "mine", Renamed: Base{ID: 2}}
	w.deepBase.Deep = "hidden"
	label("embedded").show(json.Marshal(w))
	w.Other = nil
	label("embedded-nil").show(json.Marshal(w))
	list := &Node{Value: 1, Next: &Node{Value: 2, Next: &Node{Value: 3}}, Kids: []*Node{{Value: 4}, nil}}
	label("recursive").show(json.Marshal(list))
	label("numbers").show(json.Marshal(Numbers{N: "-1.5e3",
		F: []float64{0, negZero(), 1, -1, 0.1, 1e-6, 1e-7, 123456789, 1e20, 1e21, 1.7976931348623157e308, 5e-324, 3.141592653589793},
		G: []float32{0.1, 1e-7, 1e21, 3.4028235e38, 16777216},
		I: []int64{-9223372036854775808, 9223372036854775807},
		U: []uint64{18446744073709551615}}))
	label("bad-number").show(json.Marshal(Numbers{N: "1e"}))
	label("empty-number").show(json.Marshal(json.Number("")))
	zero := 0.0
	_, err := json.Marshal([]float64{1 / zero})
	var uv *json.UnsupportedValueError
	println("inf", errors.As(err, &uv), err.Error())
	_, err = json.Marshal(float32(-1 / zero))
	println("neg-inf", err.Error())
	cycle := &Node{Value: 1}
	cycle.Next = cycle
	_, err = json.Marshal(cycle)
	println("cycle", errors.As(err, &uv), err.Error())
}

func negZero() float64 {
	z := 0.0
	return -z
}
