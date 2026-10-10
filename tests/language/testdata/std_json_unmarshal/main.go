package main

import (
	"github.com/eugenioenko/goalchemy/std/encoding/json"
	"github.com/eugenioenko/goalchemy/std/encoding/jsonvalue"
	"github.com/eugenioenko/goalchemy/std/errors"
	"github.com/eugenioenko/goalchemy/std/strconv"
)

type Inner struct {
	N  int
	S  string
	L  []Inner
	M  map[string]Inner
	PI *Inner
}

type Emb struct{ E int }

type T struct {
	Name  string
	Age   int8
	U     uint
	F     float32
	B     bool
	P     *int
	In    Inner
	IP    *Inner
	Sl    []int
	Ar    [2]int
	M     map[string]int
	MI    map[int]string
	Any   any
	Bytes []byte
	Q     int64  `json:",string"`
	QS    string `json:"qs,string"`
	Emb
	Ren int `json:"renamed"`
}

type Celsius float64

func (c *Celsius) UnmarshalJSON(b []byte) error {
	f, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		return errors.New("celsius: " + string(b))
	}
	*c = Celsius(f + 273.15)
	return nil
}

type Wrap struct {
	C   Celsius
	CP  *Celsius
	Doc jsonvalue.Value
	Raw json.RawMessage
	Num json.Number
}

func describe(err error) string {
	if err == nil {
		return "<nil>"
	}
	s := err.Error()
	var ute *json.UnmarshalTypeError
	if errors.As(err, &ute) {
		s += " | " + ute.Value + " | " + ute.Type + " | " + strconv.FormatInt(ute.Offset, 10) + " | " + ute.Struct + " | " + ute.Field
	}
	var se *json.SyntaxError
	if errors.As(err, &se) {
		s += " @" + strconv.FormatInt(se.Offset, 10)
	}
	var ie *json.InvalidUnmarshalError
	if errors.As(err, &ie) {
		s += " invalid(" + ie.Type + ")"
	}
	return s
}

func encoded(b []byte, err error) string {
	if err != nil {
		return "marshal: " + err.Error()
	}
	return string(b)
}

func main() {
	cases := []string{
		`{"Name":"a","Age":5,"U":1,"F":1.5,"B":true,"P":7,"In":{"N":1},"IP":{"S":"x"},"Sl":[1,2],"Ar":[1,2,3],"M":{"a":1},"MI":{"3":"x"},"Any":{"x":[1,"s",null,true]},"Bytes":"aGk=","Q":"12","qs":"\"s\"","E":4,"renamed":9}`,
		`{"Name":"a","Age":}`,
		`{"Name":1,"Age":5,"U":-1,"F":"x","B":2}`,
		`{"Age":300}`, `{"Age":1.0}`, `{"Age":1e2}`, `{"F":1e40}`,
		`{"NAME":"x","name":"y","nAmE":"z"}`, `{"Zzz":1,"Name":"n"}`, `{"Name":"a","Name":"b"}`,
		`{"Name":null,"Age":null,"P":null,"In":null,"IP":null,"Sl":null,"M":null,"Any":null,"Bytes":null,"Q":null}`,
		`{"Sl":[]}`, `{"Ar":[9]}`, `{"Bytes":"!!"}`, `{"Bytes":[1,2]}`, `{"Q":12}`, `{"Q":"x"}`, `{"Q":"null"}`,
		`{} x`, ``, `  `, `[1]`, `{"In":{"N":"x"}}`, `{"Sl":[1,"x",3]}`, `{"M":{"a":"x","b":2}}`, `{"MI":{"k":1}}`,
		`{"In":{"L":[{"M":{"z":{"PI":{"N":true}}}}]}}`, `{"Ar":[1,{}]}`, `{"E":"s"}`, `{"renamed":"s"}`,
		`{"Sl":["a"],"M":{"k":"b"}}`, `{"In":5}`, `{"In" : { "N" :  "x"  } }`, "{\"Name\":\"a\xffb\\ud800\\u00e9\"}",
		`{"qs":"s"}`, `{"qs":12}`, `{"B":"true"}`, `{"Any":1e400}`, `[1 2]`, `{"a":1,}`, `nul`, `1.`, "{\"KKY\":1}",
	}
	for i, c := range cases {
		t := &T{Name: "keep", Sl: []int{7, 8, 9}, M: map[string]int{"old": 1}, Ar: [2]int{5, 6}}
		err := json.Unmarshal([]byte(c), t)
		println(i, describe(err), encoded(json.Marshal(t)))
	}

	var w Wrap
	err := json.Unmarshal([]byte(`{"C": 25, "CP": 1e2, "Doc": {"b":[1, 2.50], "a":null}, "Raw": [ 1 , 2 ], "Num": "12.5"}`), &w)
	println("wrap", describe(err), encoded(json.Marshal(&w)))
	err = json.Unmarshal([]byte(`{"C": "hot", "Num": "x", "Doc": {"a":1,"a":2}}`), &w)
	println("wrap-errors", describe(err))

	var a any
	err = json.Unmarshal([]byte(`{"a":[1,{"b":null}],"c":1.5,"d":"é"}`), &a)
	println("any", describe(err), encoded(json.Marshal(a)))
	var m map[string]any
	err = json.Unmarshal([]byte(`{"a":1}`), &m)
	println("map", describe(err), encoded(json.Marshal(m)))
	var sl []int
	err = json.Unmarshal([]byte(`[1,"x"]`), &sl)
	println("top-slice", describe(err), encoded(json.Marshal(sl)))
	var n int
	err = json.Unmarshal([]byte(`"x"`), &n)
	println("top-int", describe(err))
	println("nil", describe(json.Unmarshal([]byte(`1`), nil)))
	var np *T
	println("nil-ptr", describe(json.Unmarshal([]byte(`1`), np)))
	var holder any = &m
	err = json.Unmarshal([]byte(`{"z":[true]}`), holder)
	println("holder", describe(err), encoded(json.Marshal(m)))
	var nonPointer any = 5
	println("non-pointer", describe(json.Unmarshal([]byte(`1`), nonPointer)))
	var unsupported any = &T{}
	println("unsupported", describe(json.Unmarshal([]byte(`{}`), unsupported)) != "")
}
