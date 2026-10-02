package encoding

import (
	"strings"
	"testing"
)

func TestRFC4648(t *testing.T) {
	for _, v := range [][2]string{{"", ""}, {"f", "Zg=="}, {"fo", "Zm8="}, {"foo", "Zm9v"}, {"foob", "Zm9vYg=="}, {"fooba", "Zm9vYmE="}, {"foobar", "Zm9vYmFy"}} {
		s, e := Base64Encode([]byte(v[0]))
		if e != nil || s != v[1] {
			t.Fatal(s, e)
		}
		b, e := Base64Decode(v[1])
		if e != nil || string(b) != v[0] {
			t.Fatal(b, e)
		}
		s, e = Base64URLEncode([]byte(v[0]))
		if e != nil || s != strings.TrimRight(v[1], "=") {
			t.Fatal(s, e)
		}
		b, e = Base64URLDecode(s)
		if e != nil || string(b) != v[0] {
			t.Fatal(b, e)
		}
	}
	s, e := Base64URLEncode([]byte{251, 255})
	if e != nil || s != "-_8" {
		t.Fatal(s, e)
	}
}
func TestStrictRejection(t *testing.T) {
	for _, s := range []string{"Zg", "Zg=", "Zh==", "Zg===", "Zg==\n", " Zg==", "Zm9v\r\n", "-_8="} {
		if _, e := Base64Decode(s); e == nil {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"Zg==", "Zh", "+/8", "-_9", "Zg\n"} {
		if _, e := Base64URLDecode(s); e == nil {
			t.Fatal(s)
		}
	}
}
