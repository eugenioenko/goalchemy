package main

import (
	"github.com/eugenioenko/goalchemy/std/encoding/jsonvalue"
	"github.com/eugenioenko/goalchemy/std/errors"
	"github.com/eugenioenko/goalchemy/std/strconv"
	"github.com/eugenioenko/goalchemy/std/strings"
)

func show(label string, data string) {
	v, err := jsonvalue.Parse([]byte(data))
	if err != nil {
		println(label, "error:", err.Error())
		return
	}
	out, err := jsonvalue.Encode(v)
	if err != nil {
		println(label, "encode error:", err.Error())
		return
	}
	println(label, v.Kind().String(), string(out))
}

func main() {
	doc := `{"sub":"alice","exp":1735689600,"big":123456789012345678901234567890,"ratio":0.25,
		"roles":["admin","dev"],"nbf":null,"ok":true,"nested":{"a":[1,{"b":"é𝄞"}]}}`
	v, err := jsonvalue.Parse([]byte(doc))
	if err != nil {
		panic(err)
	}
	println(v.Kind().String(), v.Len(), strings.Join(v.Keys(), ","))
	println(v.Get("sub").String(), v.Get("exp").Int64(), v.Get("big").Number(), v.Get("ratio").Float64() == 0.25)
	println(v.Get("roles").Index(0).String(), v.Get("roles").Len(), v.Get("roles").Index(5).Exists())
	println(v.Get("nbf").IsNull(), v.Has("nbf"), v.Has("missing"), v.Get("ok").Bool(), v.Get("missing").Get("x").Int64())
	println(v.Get("nested").Get("a").Index(1).Get("b").String(), v.Get("nested").String())
	if _, ok := v.Get("big").AsInt64(); !ok {
		println("big does not fit int64")
	}
	if f, ok := v.Get("big").AsFloat64(); ok {
		println("big as float64", strconv.FormatFloat(f, 'g', -1, 64))
	}
	for i := 0; i < v.Get("roles").Len(); i++ {
		r := v.Get("roles").Index(i)
		s, _ := r.AsString()
		println("role", i, s)
	}

	built := jsonvalue.Object(
		"sub", jsonvalue.String("<b>&\"x\"\n "),
		"exp", jsonvalue.Int(-9223372036854775808),
		"max", jsonvalue.Uint(18446744073709551615),
		"floats", jsonvalue.Array(jsonvalue.Float(0.1), jsonvalue.Float(1e21), jsonvalue.Float(1e-7), jsonvalue.Float(-0.0), jsonvalue.Float(123456789)),
		"exact", jsonvalue.Number("1.10"),
		"none", jsonvalue.Null(),
	)
	built = built.Set("exp", jsonvalue.Int(42)).Set("extra", jsonvalue.Array().Append(jsonvalue.Bool(false)))
	out, err := jsonvalue.Encode(built)
	println(string(out), err == nil)
	println(built.String())
	println(jsonvalue.String("plain").String(), jsonvalue.Int(7).String(), jsonvalue.Null().String())

	show("ws", " \t\n[ 1 , \"two\" , { } , [ ] ]\r\n")
	show("escapes", `"\"\\\/\b\f\n\r\tA\u0000"`)
	show("lone", `["\uD834","\uDD1E","\uD834A"]`)
	show("utf8", "\"ok\xff\xe2\x82\xac\xed\xa0\x80\"")
	show("numbers", `[0,-0,1.5e+308,1e400,-2.5E-3,9007199254740993]`)
	show("trailing", `[1] x`)
	show("comma", `[1,]`)
	show("dup", `{"a":1,"a":2}`)
	show("leading zero", `01`)
	show("control", "\"a\x01\"")
	show("escape", `"\x"`)
	show("literal", `nul`)
	show("empty", ``)
	show("member", `{1:2}`)
	show("depth", strings.Repeat("[", 128)+strings.Repeat("]", 128))
	show("too deep", strings.Repeat("[", 129)+strings.Repeat("]", 129))

	l := jsonvalue.DefaultLimits()
	l.StringBytes = 4
	_, err = jsonvalue.ParseWithLimits([]byte(`["abcd","abcde"]`), l)
	var limit *jsonvalue.LimitError
	println(errors.As(err, &limit), limit.Limit)
	_, err = jsonvalue.Encode(jsonvalue.Number("1."))
	var invalid *jsonvalue.InvalidValueError
	println(errors.As(err, &invalid), err.Error())
	nan := jsonvalue.Float(0)
	zero := 0.0
	nan = jsonvalue.Float(zero / zero)
	_, err = jsonvalue.Encode(nan)
	println(err.Error())
	_, err = jsonvalue.Parse([]byte(`{"a":[1,}`))
	var syntax *jsonvalue.SyntaxError
	println(errors.As(err, &syntax), syntax.Offset, err.Error())
}
