package main

import (
	"github.com/eugenioenko/goalchemy/std/encoding/json"
	"github.com/eugenioenko/goalchemy/std/encoding/jsonvalue"
)

type Envelope struct {
	Kind string          `json:"kind"`
	Body jsonvalue.Value `json:"body"`
	Meta any             `json:"meta,omitempty"`
}

func main() {
	doc, err := jsonvalue.Parse([]byte(`{"z":1,"a":["<x>",2.50,true,null],"n":{}}`))
	println(err == nil)
	b, err := json.Marshal(Envelope{Kind: "doc", Body: doc, Meta: []any{doc, jsonvalue.String("&")}})
	println(string(b), err == nil)
	b, err = json.MarshalIndent(Envelope{Kind: "built", Body: jsonvalue.Object("k", jsonvalue.Array(jsonvalue.Int(1), jsonvalue.Float(0.5)))}, "", "  ")
	println(string(b), err == nil)
	_, err = json.Marshal(Envelope{Kind: "invalid"})
	println(err.Error())
	_, err = json.Marshal(jsonvalue.Number("NaN"))
	println(err.Error())
}
