package main

import (
	"github.com/eugenioenko/goalchemy/std/encoding/json"
	"github.com/eugenioenko/goalchemy/std/errors"
	"github.com/eugenioenko/goalchemy/std/strconv"
)

type Text int

func (t *Text) UnmarshalText(b []byte) error {
	if string(b) == "bad" {
		return errors.New("bad text")
	}
	*t = Text(len(b))
	return nil
}

func (t Text) MarshalText() ([]byte, error) { return []byte("t" + strconv.Itoa(int(t))), nil }

type JS struct{ Raw string }

func (j *JS) UnmarshalJSON(b []byte) error {
	if string(b) == `"fail"` {
		return errors.New("js fail")
	}
	j.Raw = string(b)
	return nil
}

type Shape interface{ Area() int }

type Leaf struct {
	I   int
	I8  int8
	U16 uint16
	U   uint
	F32 float32
	F64 float64
	S   string
	B   bool
	Bs  []byte
	By  [2]byte
}

type Opt struct {
	O string
	I int
}

type Rec struct {
	Leaf
	*Opt
	Name   string  `json:"name"`
	Q      int     `json:",string"`
	QS     string  `json:",string"`
	QB     bool    `json:",string"`
	QF     float64 `json:",string"`
	QP     *int    `json:",string"`
	P      *int
	PP     **string
	Kids   []Rec
	A      [3]int
	M      map[string]int
	MI     map[int8]string
	MU     map[uint]bool
	MT     map[Text]int
	Any    any
	AL     []any
	AM     map[string]any
	T      Text
	TP     *Text
	J      JS
	JP     *JS
	Raw    json.RawMessage
	Num    json.Number
	Shape  Shape
	Nested *Rec
}

func describe(err error) string {
	if err == nil {
		return "ok"
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
	return s
}

func encoded(b []byte, err error) string {
	if err != nil {
		return "marshal: " + err.Error()
	}
	return string(b)
}

func prefilled() *Rec {
	seven := 7
	kids := make([]Rec, 2, 4)
	kids[0].Name = "k0"
	kids[1].Name = "k1"
	return &Rec{Name: "keep", Kids: kids, A: [3]int{1, 2, 3}, M: map[string]int{"old": 1}, P: &seven, Opt: &Opt{O: "o"},
		Nested: &Rec{Name: "n"}, Any: "old", AM: map[string]any{"old": true}, AL: []any{"x", "y", "z"}, T: 9}
}

func main() {
	for i, doc := range docs {
		var fresh Rec
		err := json.Unmarshal([]byte(doc), &fresh)
		println(i, "fresh", describe(err), encoded(json.Marshal(&fresh)))
		pre := prefilled()
		err = json.Unmarshal([]byte(doc), pre)
		println(i, "pre", describe(err), encoded(json.Marshal(pre)))
		var a any
		err = json.Unmarshal([]byte(doc), &a)
		println(i, "any", describe(err), encoded(json.Marshal(a)))
		var inner any
		var holder any = &inner
		err = json.Unmarshal([]byte(doc), holder)
		println(i, "holder", describe(err), encoded(json.Marshal(inner)))
		var js JS
		err = json.Unmarshal([]byte(doc), &js)
		println(i, "js", describe(err), js.Raw)
		var leaves []Leaf
		err = json.Unmarshal([]byte(doc), &leaves)
		println(i, "leaves", describe(err), encoded(json.Marshal(leaves)))
		var tm map[Text]*Text
		err = json.Unmarshal([]byte(doc), &tm)
		println(i, "textmap", describe(err), encoded(json.Marshal(tm)))
	}
}

var docs = []string{
	"[\t \"-2.5\"]",
	"{\t \"a/b\":\"\",\"O\": \"\"}",
	"\t {\"PP\"\t :\ntrue,\r\n  \"QS\":1e40,\"I8\":{\"1e2\":\r\n  true,\"QF\":\t true,\"QS\":{ \"U16\":{\"B\": \"aGk=true\"}}\t },\"MI\":5e-324}",
	"{\n\"QB\":\"true\"\t ,\"S\"\n:null,\"MI\":{\"Opt\":null\n,\n\"TP\"\n:\"\x01\"}\r\n  ,\n\"MI\":{\"QS\":127,\r\n  \"JP\":9223372036854775807}} ",
	"0.1",
	"{\"MI\" :null,\"a/b\":{\"Any\":\"\", \"S\"\r\n  :\"Z\\t\",\"P\" :\"YQ=\\t\"\n,\r\n  \"PP\":\"YQ=\"}\r\n  ,\n\"U\":\"<&>😀😀\"\r\n  ,\r\n  \"MU\": -129}",
	"\t {\"MI\":{ \"J\":false\n,\"unknown\":123456789012345678901234567890,\"k\":\"é\"\t ,\"Nested\":{}}} ",
	"{\"MT\"\t :true ,\n\"MT\":{\t \"U\":null,\"JP\"\n:null,\t \"U16\":[]\n,\"+2\"\n:\"\\t\\udc00x\"},\r\n  \"MU\"\n:null,\"I8\":false}",
	"\r\n  {\t \"TP\"\t : {}\n,\"MI\":[\"1truetrue\", {\"01\":\r\n  {\"NAME\":-1.5,\"1e2\":\n\"!!0abc\"\r\n  }},\r\n  { \"I8\"\n:\"\"}]\t }",
	"\r\n  {\"Ñ\"\r\n  :[5e-324,{} ,\r\n  65535] ,\"AM\":[\"\\ud800\",\" YQ=0\"],\"x~y\":{},\n\"K\":\t \"\xff\\ud800null\" }",
	"\n{}",
	"{\"1\":\t -129}",
	"{\r\n  \"300\":null,\"Ñ\":\r\n  1e-400\r\n  ,\"q\":\t [\r\n  {\r\n  \"NAME\"\r\n  :null,\"F32\"\t :null,\"MT\":true,\"300\"\n:\"😀\"},3.4028235e38\n,\"A 0\"],\"JP\": {\"Raw\":-1\n,\"Num\":\"A\",\"QP\":{ \"unknown\":\"!!1😀\",\"Bs\":{},\"f64\":5e-324\n}\n}\t }\t ",
	"{\"1\":\"AQID<&>\\ud800\",\"300\" :\n\"\", \"Shape\"\r\n  :{\"01\" :{\"QP\":{\"K\"\t :true,\"NAME\":\"AQID1\"\r\n  ,\"nested\":\"\",\"nested\":\r\n  null}\t },\"Num\":\n123456789012345678901234567890,\t \"1e2\":\"é\\/\\t\", \"bad\":[[],null]}}",
	"{\"Ñ\":{\n\"Shape\"\n:\rE\n  {\r\n  \"M\":\"YQ===\"}\t ,\"1e2\":65535,\"1\":1e400},\"MU\":{\"NAME\": \"failZfail\"\n,\"TP\":2.5e-3,\"K\":-1e400,\"i\" :\t {\n\"nested\":\"\\udc00x\", \"QS\":null\t }}}",
	"{\"Q\":{\"a/b\" :{\"J\":-1,\"A\":{\"O\":\"\"\n,\"F64\":-1e400,\"kids\":\"\",\"U16\":1e40},\"Name\":\t \"\\ntrue!!\",\"i\":false} ,\"K\"\n:\t [\"\\n\",{\"QP\"\n:18446744073709551616,\"O\":false,\"Num\":\"\"\r\n  ,\"01\"\n:\r\n  123456789012345678901234567890},true ]\t ,\"U16\":\" \"},\"F64\":{\"NAME\":65535\r\n  }}",
	"false",
	"[\n{},true]",
	"{\"+2\":null,\"1\":1e40}\n",
	"\n{\"AL\":{\n\"B\":\"A\\\"fail\\\"\"},\t \"P\"\r\n  :\"0\\ud800\\\"\"\t ,\"Nested\" :null}",
	"8[1e2",
	"{ \"QP\"\r\n  :-1e400}",
	"\n{\r\n  \"S\":-1.5}",
	"{\"U\":null,\"Num\":\"1\\n\\/\",\r\n  \"Shape\":\"\\\"\"}\t ",
	"{}",
	"{\"F32\":\r\n  \"\"\n,\"Kids\":null,\n\"MU\":\"\",\"TP\":{}}",
	"9223372036854775808",
	"nul",
	"{\t \"a/b\":{\n\"Q\":2.5e-3,\r\n  \"PP\":false,\"NAME\":1e400},\"P\":true,\"AL\":{\"name\":{\"Name\":\"YQ=\\\"s\\\"bad\",\"J\":18446744073709551616,\"I\":65536\r\n  },\"AL\":null}}",
	"{\"AL\"\t :\r\n  {\"A\":1e40 ,\"K\":[\t {\"MU\":\r\n  \"\",\"AM\": 1E-2}]}\r\n  }",
	"false",
	"{\"J\":\nfalse}",
	" {\"bad\":\t null,\"Nested\":[{\"-3\"\n:\"\",\"Nested\":[]},5e-324\n]}",
	"{\"F64\":\t {},\"MI\":[1e-400]}",
	"{} ",
	" {\"+2\":false\t ,\"O\":\r\n  {},\"i\":\"é\",\"T\":\"\"}",
	"{\"MT\": {\"unknown\":\n\"\",\"TP\"\n:{\"-3\" :\"é<&>\"\n,\"NAME\":\n\"aGk=\r\n\\\\\",\"TP\":[5e-324,\"\xff\",\"\\tYQ===\" ],\t \"AL\":\"\"}, \"f64\":\"YQ\\n\"},\r\n  \"name\" :null} ",
	"[9223372036854775808]",
	"\r\n  {\t \"1e2\":\n\"\" ,\"QS\"\n: {\"J\": \"\x01\",\"B\":\n255\t ,\"Bs\":\"\\\\\\ud800\\t\"},\"A\"\n:true,\" 4\"\r\n  :\ntrue}\t ",
	"{\"QB\":[{\"QB\":{\t \"QP\"\r\n  :false}},{ \"Bs\":\n{\"nfsted\":65535,\n\"U16\":false,\"q\":1.0},\"Q\":{\"Raw\":null}} ],\t tPP\":false\r\n  ,\"AM\":{\"Opt\":18446744073709551615,\"A\":true,\"f64\"\r\n  :null,\r\n  \"M\":\"\\\"fail\\\"\"}}",
	"{\"S\":true,\"i\":-129\r\n  ,\"J\":\r\n  false}",
	"false",
	"{Num\":\"aGk=\r\n\"}",
	"\r\n  {\"01\":\r\n  null,\"+2\": 65536\n}",
	"\n{}",
	"{\"Q\":[\"é\",null ],\"JP\":null,\t \"Nested\":\"YQ===\"}",
	"{}",
	"{\t \"Opt\":\"\\udc00x\"}",
	"{\"Raw\":[1.0 ,{\r\n   \"QP\":\t {},\n\"a/b\":[1E-2,\n\"😀éaGk=\",\t null],\"Bs\":null,\t \"QS\":null}],\t \"T\":{},1\":1E-2,\"kids\":false}",
	"{\"nested\":true, \"Nested\":null,\"MT\":null}",
	"\r\n  {\"U16\"\r\n  :{\"F32\": null },\"I8\" :\"é\"\n,\"U16\":{\"1e2\"\r\n  :\n{\"F32\":\"YQ=\\\"fail\\\"\"},\"J\"\n:\"\\\"fail\\\"-2.5\",\"a/b\":{\"Name\":{}, \"B\" :\"\",\"300\":\t {\"J\":\"\\udc00x0\"}},\t \"F64\":{\r\n  \"MT\":\"\\\"fail\\\"\"}},\"k\":true}",
	"{\"Ñ\": [{\"PP\"\r\n  :\r\n  {\"Name\":\r\n  \"1A\",\"M\" :\"aGk=\r\n<&>\"},\"U16\"\n:[\r\n  false]},\"YQé\\\\\",{} ],\"300\":{}}",
	"\t {}",
	"{ \"MT\" :65535 ,\"1e2\":-0}",
	"null",
	"{\"S\":{\"T\"\n:\"\\t-2.5\",\"I\":\"\"},\"Raw\":{\"Bs\":{\t \"AL\": {\"Kids\"\r\n  :\"\",\t \"-3\":-1e400\n,\"AM\"\r\n  :65536}}},\"bad\":128}\t ",
	"{\"x~y\":{},\"Opt\":65535,\"name\":\"0😀YQ===\",\"Bs\":\r\n  \"é\x01<&>\"}",
	"{}",
	"{\"Nae\":\"\\u0c00x\\\"\"\n,\"I\"\n:{\n\"MI\":\n[null,127]\r\n  ,\"By\" :\r\n  true}}",
	"{\n\"T\":{\"O\":\"\"},\t \"a/b\":\r\n  \"\\\"s\\\"0\xff\",\"B\":{\n\"300\" :18446744073709551615,\n\"k\":{},\"U16\": {\t \"I\":\"\\né\"\r\n  }},\"1e2\"\r\n  :\"true\\udc00x\\\"\"}",
	"{\"S\":{\"MT\":null,\"Ñ\":\t {},\"+2\":\"\",\"Bs\":65535},\"I\":\n65536,\"AL\"{\"QF\":[ -1,{\"QS\"\n:\"\\nZ\"\"A\":-1.5,\"x~y\":\"\",\"MI\":\r\n  \"YQ=-2.5\"}]}}",
	"\r\n  {\"i\":null\t ,\"Num\":{\"Name\":{\"I\":\r\n  1e400,\"AL\":[1e40,\"\\\"fail\\\"YQ===a\",\t \" \xff\"]},\"F64\":256,\"AL\": true\n}}",
	"{\"name\":\r\n  1.0,\"i\"\t :0,\n\"A\":null}",
	"\r\n  {\"Kids\":3.4-282+5e38}\n",
	"{\t \"Opt\":{\n\"Ñ\":1e2,\t \"I8\"\r\n  :true},\"k\":[\"AYQ===A\"\t ,{} ],\"U16\":\t 1e2,\n\"U\": {\"F32\":{}\n,\n\"JP\"\t :\nnull ,\"K\":\r\n  {\" 4\":\" 😀\" ,\t \"300\":\"\",\"1e2\"\n:0.1},\"f64\" :\t 1e2}}",
	"{\r\n  \"-3\":\"0<&>YQ\" ,\"Any\":{\"Kids\"\t :false,\"A\":\n2.5e-3\r\n  ,\"QP\":\r\n  null\n},\"Opt\":null,\"Name\":\n\"a\"\r\n  }\r\n  ",
	"{ \"Num\"\t :\"\\\"s\\\"\\\"\"}",
	"false",
	"\n{\"Shape\":\"\x01A\\t\",\"Raw\":18446744073709551615,\"f64\": null ",
	"{\"P\":\"\",\"Kids\":{\"MU\":\r\n  18446744073709551616\n,\r\n  \"1\":\"true\",\"NAME\":{\"Any\":\n[null,\"true\",\t false],\"F32\":18446744073709551616,\"1e2\":\"-2.50aGk=\"} ,\"Bs\": {}}}\r\n  ",
	"\r\n  {\r\n  \"q\":-129\t }",
	"false",
	"{ \"By\":false,\"By\":5e-324} ",
	"{\"P\":\r\n  { \"MT\":{\"nested\":{\"1e2\":\"\"\n, \"unknown\"\n:null,\"S\":\"aGk=\r\n\\\"fail\\\"\"\t }\t ,\t \"Q\":null\r\n  ,\"O\":3.4028235e38,\"A\":{\"TP\":-0,\"q\": null}\t } ,\"MI\"\r\n  :{ \"O\"\n:9223372036854775807,\"1e2\":{\"U16\":\"\\\"s\\\"\"\n,\r\n  \"f64\" :\t false,\"01\":\"\"},\"k\":1}\n}, \" 4\":123456789012345678901234567890,\r\n  \"Opt\":{\"Raw\":1e2,\"Shape\":1E-2,\t \"U16\":3.4028235e38,\"q\":[2.5e-3,\"!!YQ=== \"]} ,\"I\" :true\r\n  }",
	"{\"MT\": \"\"\t ,\n\"U\":1} ",
	"\"\\\"s\\\"-2.5\"",
	"{\"k\":[\r\n  \"\",{\"U16\" :null,\"JP\"\t :\r\n  {\"O\"\n:\"\"},\"f64\"\r\n  :\"failnullnull\",\n\"Kids\":false}],\r\n  \"TP\":\"\",\"x~y\":65535\n,\t \"K\":\r\n  -1e400}\r\n  ",
	"{}",
	"{\"P\": {\t \"name\"\r\n  :\r\n  {\"QS\"\t :\r\n  null,\" 4\":{\"Name\":null,\"QB\" :\t \"abcé\",\"AL\"\n:\"aGk=YQ\\\"s\\\"\"\r\n  }},\"U\":{},\r\n  \"01\":{\" 4\":null},\"300\":\r\n  \"\" }}",
	"{\"NAME\":128,\"B\"\n  :\"\", \"Kids\":1e-400}",
	"\r\n  {\"B\":[\r\n  [true\r\n  ]\n,[\t \"A\\udc00xnull\"]\n,{}],\"J\":[ 1e-400,null\t ],\r\n  \"AL\"\t :[]\r\n  }",
	"{\"Nested\":\r\n  \"\",\r\n  \"QF\":\"fail0\"\r\n  ,\"By\":true\t ,\"AM\"\t :\"trueafail\"}\t ",
	" {} ",
	"{\"AL\"\r\n  :\r\n  false,\r\n  \"unknown\":\"null\\ud800<&>\", \"U16\"\t :{\" 4\"\n:\t -1e400,\"Name\":[\"AQID\" ] },\"01\" :[\n[]\t ,\r\n  3.4028235e38,\n\"YQ-2.5\"]\n}\r\n  ",
	"\r\n  {}\r\n  ",
	"{\"I8\" :\r\n  \"\\n\"}",
	"{\"01\":\"\\\\\\\"\",\"O\"\n:65535, \"M\":\"\",\"NAME\":\"YQ===\" }",
	"{\"F64\":false}",
	"{\t \"Opt\":{\"PP\":true},\"F32\":null}\n",
	"{}",
	"1E-2",
	"{\"bad\":false,\"PP\": {\"TP\":null,\"A\":null\t ,\"Raw\"\r\n  :\n{\"kids\":-1e400\n,\"PP\"\r\n  :1e40 ,\"Ñ\":\"\\udc00x\",\"Name\":\t true }},\"Raw\":[\r\n  false, \"YQnullAQID\"]}",
	"{}",
	"{\"Opt\":{\"1e2\" :\n[\"\xffaGk=\"]}\t }",
	"{}",
	"{\r\n  \"By\":{\t \"K\" :{\n\"AL\":{}\r\n  ,\"QS\"\t :[]\t ,\"M\"\r\n  :\ntrue},\"JP\":{\"Any\" \"YQ\\t\xff\"}}}",
	"-1.5",
	"\r\n  {}\r\n  ",
	"{}\n",
	"t\"\xf0\\\x9f\x98\x80\"",
	"{\"bad\":{},\"a/b\"\t :{\"+2\":null},\"M\": \"-2.5\"}",
	"{\"i\"\n:\"aAQID\"\r\n  ,\"MT\" : \"\\udc00x\",\"Any\":\n{\"QP\":false ,\"MU\":\"YQ\\\"fail\\\"\"\t },\t \"MI\":{\"kids\":\"\",\"unknown\":\n{\"AM\":{}\n,\t \"JP\":\"Z\",\"AM\":null,\"QB\":5e-324\t }}}",
	"{}",
	"1234567890124567890123567890",
	"{\r\n  \"TP\"\t :{\"U16\":\"\\\"s\\\"\\\"fail\\\"😀\"},\t \"M\":{\"T\"\n:null},\"NAME\":{\n\"T\"\t :\r\n  {\"1\":\"\",\" 4\":1.0\n,\"-3\":255},\"Name\":false,\"k\":\n-1e400}\t ,\"K\":\t { \"Q\":\"\"\r\n  ,\t \"QF\"\r\n  :null,\"F64\" :{}\r\n  }}",
	"{}\r\n  ",
	"{\"Num\":\t null}",
	"{\"name\":true,\r\n  \"q\" :true,\" 4\":\"fail\\\\\"}",
	"{\r\n  \"P\":1e00 ,\"AL\":{\"O\":-1e400}}",
	"{ \"P\":false,\"Shape\":\r\n  [],\"I8\":\r\n  -129,\"a/b\": -1}\t ",
	"{\"-3\":\"\",\"AL\":\"1\"\r\n  ,\r\n  \"Any\"\r\n  : \"\"}",
	"\r\n  {\"x~y\"\r\n  :127,\"x~y\":\t \"YQ\\ud800YQ===\" }",
	"{\"P\":{\"S\":{\"unknown\":-1.5,\"I8\"\n:\t {\"I\":\n\"\",\"Any\": 1E-2,\"F32\":false},\"nested\": {\"x~y\" :\"\\t\",\"01\":\"aAZ\",\"x~y\"\r\n  :\n\"\xffYQ===é\",\"f64\"\r\n  :true\t }},\"kids\":false,\"JP\":\n\"\"}\n,\"K\"\n:true ,\"x~y\":\"Z\\t\",\"JP\"\n:-1e400\t }",
	"{\t \"-3\":\"1\\\\\xff\",\"x~y\" :-1.5\t ,\"J\": {\"1e2\":null,\t \"QB\":\"\",\"QB\" :\"Atrue!!\",\"01\":false},\"T\":null}",
	"256",
	"{}",
	"{\r\n  \"Nested\":\"aGk=\r\n\"}",
	"\r\n  {\"MI\":\t 18446744073709551616}\n",
	"{\n\"q\":{}}",
	"{\"q\"\n:[\"\\udc00x\\t<&>\",\t -0],\"kids\":\"\\\"fail\\\"\",\n\"Num\":{},\"NAME\"\n:{}}",
	"{\"01\"\r\n  :\"a\"}",
	"{ \"B\" :\"\"}",
	"{}\n",
	"\"Z\"",
	"65535",
	"{\"QS\":\t -129,\"O\":2.5e-3\t ,\t \"By\":0.1}",
	"{ \"x~y\"\n:{\"MU\":1-e40\n}\n}",
	"\n{",
	"{\"Raw\":{\"Raw\":\" \"\t ,\"x~y\"\t :\" -2.5\",\"O\":{\"MU\"\t :1e-400\t }\r\n  ,\r\n  \"K\":256},\" 4\":0,\"MI\":\t {\n\"Opt\"\r\n  :[\r\n  {}],\"Name\":1,\"01\": null\r\n  ,\n\"name\":false},\"PP\"\t :\"\"} ",
	"{\"O\"\n: {\"MI\":\r\n  \"\\ud800\"},\r\n  \"+2\":true}",
	"{\"bad\":\"ab!\"}",
	"{\"Shape\":null,\"a/b\"\n:\r\n  \"é\\/ \",\"MU\":null,\t \"QB\":1e2}",
	"123456789012345678901234567890",
	"{\"kids\":0,\n\"Raw\":{\"S\":\"!!\"\n,\"i\":{\n\"Shape\":1.0},\t \"Bs\":\"YQ\"\t }}",
	"{\"P\"\n:[1e400\r\n  ]}",
	" {\r\n  \"MU\":\t {\"M\":false\r\n  ,\n\"Any\" :\"\",\"Kids\":{\"U\"\r\n  :[false,1e400]\t ,\"MU\":\t 18446744073709551616,\"Name\":null}, \"MT\" :{\"Num\"\t :true}} ,\t \"Name\"\t :\"\\t\",\"q\":\nnull\r\n  }\n",
	"{}",
	"{\"T\":false,\r\n  \"S\"\n:{\"-3\"\t :\"0\",\"Name\":-0}}",
	"{}",
	"{\"Shape\":null,e\n  \"i\": {\"MT\":\r\n  256,\"Bs\xff\":null,\"S\" :1e400}}",
	"{\"S\":null,\"S\":\r\n  {\"Bs\":\"\\\"\xff\\\"fail\\\"\"},\"QB\":\r\n  \"\\t\",\"QP\":\t {\"I\":\"\"\r\n  ,\"I\":\"Z1\\ud800\",\"A\"\r\n  :{\"MU\":null,\r\n  \"M\":\t {\"NAME\"\r\n  :\t 127\r\n  ,\"+2\":127}, \"Num\"\t :1e-400,\"F32\" :null}} }\t ",
	" {\"Shape\":{\"A\":\"é-2.5\\n\",\t \" 4\":false}\r\n  ,\"S\":{}\t ,\"J\"\t :\"\"\n,\"Num\":\r\n  true}",
	"{-",
	"{\"By\":true,\"MT\": [\t [\t true,\"\\/badaGk=\"\r\n  ,\"\"],\"!!é\"\n]\r\n  ,\"Ñ\":0.1\t }",
	"\"true1<&>\"",
	"{\"1e2\":{},\"300\":true,\"TP\":\r\n  128, \"AL\":null}",
	"{\"TP\":\t \"\\té\" ,\"k\":{\"A\":\n\"anull\",\r\n  \"kids\"\r\n  :\"\x01\",\"unknown\":\n{\t \" 4\":\"null!!\"},\"Ñ\"\n:65536}, \"1e2\"\t :\"é\\udc00x\"\t ,\n\"Opt\":{\":T\":[-0]\n, \"J\":\"A\x01\" }}",
	"{}",
	"{}\n",
	"{\"Bs\":\"\",\"MI\":null,\"kids\":true }",
	"{\"Bs\"\n:\t \" 1é\",\"JP\":\"\",\"Opt\":\r\n  \"\"}",
	"{\"Ñ\":null,\"nested\":\"\",\"300\" :\"aAAQID\"}",
	"{\"U16\":-129,\"Any\"\t :[false],\"B\":-1.5,\"QB\":-1.5}\t ",
	"{eT\" :null} ",
	"\"\\té\"",
	"\"abc\"",
	"{ \"nested\":{\"unknown\":\"\\\"fail\\\"\"\r\n  ,\"NAME\":\"\\t\\ud800\xff\",\"MT\":\"\"},\"F32\":null,\"U\":2.5e-3,\"-3\":{}}",
	"-0",
	"\n{\"Bs\" :\r\n  9223372036854775808\r\n  ,\"300\"\r\n  :-1.5\n,\"Nested\":-1e400}",
	"{\"Opt\":\"\\ud800\\ud800\"\n,\r\n  \"Q\":{\"Num\"\n:null},\"Name\":127,\"Nested\":{\"Bs\":[] }}",
	"{}",
	"\r\n  {\"MI\":\"\n,\"QP\":\t \"YQ=failbad\",\"K\":1YQ===\"\r\n  ,\"a/b\":\r\n  [\r\n  null,null]}\n",
	"{}\n",
	"{\"f64\":{\"01\":1e2\t ,\"nested\":{ \"MI\"\r\n  :\t true\n,\"MU\":\n{\r\n  \"T\":\"AQID!!\",\"U\":\"é\"}},\"Raw\":[null\t ]\r\n  , \"QP\":1e2}\r\n  } ",
	"\r\nE  {}",
	"{\"Ñ\":1E-2,\"Shape\":\r\n  {\"I8\":{\"JP\":{}, \"bad\":\"abc\"\n,\"name\": true\t },\"k\":{\"Name\"\t :\"null\\/\",\"Q\"\t :\"\\\"\"},\"T\":\"é\"\t ,\"B\":null},\t \"a/b\":-1, \"M\":true\r\n  }",
	"{\"TP\":true,\"name\"\t :\"aGk=\\\\abc\", \"Kids\": {\"nested\":\"aGk=\" }\r\n  ,\"Q\": -0\n}",
	"{}",
	"{\"1\":127,\"Num\":\n{ \"By\":\"\xff😀A\",\r\n  \"Ñ\" :\r\n  1e40\n,\"name\":\n{\"PP\":-1},\"JP\":\"\\ud800\xff\"},\r\n  \"Opt\":null}",
	"8{\"bad\"\t :127 ,\"300\"\r\n  :123456789012345678901234567890,\"JP\"\"\"\t }\r\n  ",
	"{\"QF\":\"YQ=\"\t }",
	"{\"JP\":{\"Raw\":\n9223372036854775807,\"QF\" :\r\n  [{}]},\"unknown\"\r\n  :{\"M\":false ,\"MU\":true\r\n  }, \"M\":\"\",\"F32\"\r\n  :-1e400\r\n  }",
	"{\"AM\":[\"bad\"\n,\n{}],\"QS\":\n{\"QS\":null\n,\"Any\":0}\n,\"+2\":{}}",
	"{\"P\":-1.5,\"nested\":\"\"\r\n  ,\t \"1e2\":\t \"é\\/\"\r\n  }",
	"{\"k\":{\"i\"\n:\n\"😀😀A\",\"M\":\r\n  []\r\n  },\"1\" :{\t \"M\":null},\n\"1e2\":null}\r\n  ",
	"\n{\"Nested\"\r\n  :\n\"\\t\"}",
	"{\"name\":{\"Opt\":{\"-3\":0.1},\"1e2\"\n:{\"MU\":0.1,\n\"K\":{\n\"QF\":127,\"k\": \"\"}}\n,\"Opt\":\"\",\"name\":{\n\"QF\": {\t \"-3\"\n:null,\"nested\":\"\"}}\r\n  } ,\"J\":null }\t ",
	"{}\t ",
	"\n{}",
	"\"\\nafail\"",
	"{\"AL\"\t :\"0\xa9\"\t }",
	"{\"U\":\"\\udc00x\",\"AL\"\t :null,\r\n  \"q\":\r\n  \"Z\",\" 4\":{\r\n  \"F32\"\n:\"YQ===\",\"JP\":3.4028235e38}}\r\n  ",
	"{\"A\":\"\\udc00x\",\t \"Raw\":null\n}\r\n  ",
	"\r\n  {\r\n  \"P\":\"é\"}",
	"{\"-3\":-0,\"01\":null,\"TP\":\r\n  null\t , \"S\"\t :\"YQ===\xff\"\r\n  }",
	"{\"T\" : 2.5e-3}",
	"{\"Opt\":false}",
	"{\t \"QP4:{\n\"K\":0.1,\r\n  \"k\":\"é😀\"},\"Opt\":\"\\\"fail\\[\"}\n",
	"\t {\"B\"\r\n  :{\"Num\":{\"AM\":\"aGk=é\x01\",\r\n  \"Num\"\r\n  :null,\" 4\" :[9223372036854775807]}},\"1e2\":1, \"QP\" :\r\n  -1.5}",
	"{\"S\":{\"NAME\":\"AQID\",\"S\":{\"F64\":3.4028235e38\r\n  },\"T\":\"\\n1\"\r\n  }\r\n  ,\"NAME\":true,\"PP\":\n\"<&>-2.5aGk=\"\r\n  ,\"kids\":\"\"}\t ",
	"{\n{5",
	"\"\\nbad\"",
	"{}",
	"{}",
	"\r\n \"{\"QS\":\r\n  \"\"}",
	" {\"MU\":[null,{\n\"f64\":{\"Shape\":\t \"\\tA\"},\"300\":123456789012345678901234567890}],\"NAME\":\"failYQ===\\\"fail\\\"\"}",
	" {\"k\" :[]}\r\n  ",
	"\"\\udc00xaGk=\r\nA\"",
	"{\t \"I8\"\r\n  :\"\\\"0null\"\n, \"QP\":256\r\n  ,\"U16\"\t :2.5e-3 }",
	"{}",
}
