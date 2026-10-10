package main

import (
	"github.com/eugenioenko/goalchemy/std/bytes"
	"github.com/eugenioenko/goalchemy/std/encoding/json"
	"github.com/eugenioenko/goalchemy/std/errors"
)

type Out string

func (o Out) MarshalJSON() ([]byte, error) { return []byte(o), nil }

type Nest struct {
	Name string
	Out  Out
	List []Out
}

func main() {
	var se *json.SyntaxError
	var err error
	for _, out := range []Out{"[1 2]", `{"a":`, "1 2", "", "nul", "\"a\x01\"", "\"\\x\"", "{1:2}", `{"a" 1}`, `{"a":1 "b":2}`,
		"-", "--1", "1.", "1e+", "01", "[,1]", "[1,]", "\"\\u12\"", "tru", "}", " \t[ 1 , true ,null] ", "\"\xff<\"", `{"a":1,"a":2}`} {
		b, err := json.Marshal(Nest{Name: "n", Out: "0", List: []Out{"1", out}})
		if errors.As(err, &se) {
			println("syntax", string(out), se.Offset, err.Error())
		} else {
			println("syntax", string(out), string(b), err == nil)
		}
	}
	println("valid", json.Valid([]byte(`{"a":[1,2,{"b":null}]}`)), json.Valid([]byte(`{"a":}`)), json.Valid([]byte(` 1 `)), json.Valid(nil))
	var buf bytes.Buffer
	err = json.Compact(&buf, []byte(" { \"a\" : [ 1 , \"<\" ] } "))
	println("compact", buf.String(), err == nil)
	buf.Reset()
	err = json.Compact(&buf, []byte(`{"a":`))
	println("compact-bad", buf.Len(), err.Error())
	buf.Reset()
	err = json.Indent(&buf, []byte(`{"a":[1,{}],"b":[]}`), "> ", "\t")
	println("indent-fn", buf.String(), err == nil)
	buf.Reset()
	err = json.Indent(&buf, []byte(`[1 2]`), "", " ")
	println("indent-bad", err.Error())
	buf.Reset()
	json.HTMLEscape(&buf, []byte("{\"<a>\":\"& \"}"))
	println("html-escape", buf.String())
}
