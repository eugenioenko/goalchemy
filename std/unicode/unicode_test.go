package unicode_test

import (
	"testing"
	stdunicode "unicode"

	"github.com/eugenioenko/goalchemy/std/unicode"
)

func TestMatchesStd(t *testing.T) {
	type pred struct {
		name string
		got  func(rune) bool
		want func(rune) bool
	}
	preds := []pred{
		{"IsLetter", unicode.IsLetter, stdunicode.IsLetter},
		{"IsUpper", unicode.IsUpper, stdunicode.IsUpper},
		{"IsLower", unicode.IsLower, stdunicode.IsLower},
		{"IsTitle", unicode.IsTitle, stdunicode.IsTitle},
		{"IsDigit", unicode.IsDigit, stdunicode.IsDigit},
		{"IsNumber", unicode.IsNumber, stdunicode.IsNumber},
		{"IsPunct", unicode.IsPunct, stdunicode.IsPunct},
		{"IsSymbol", unicode.IsSymbol, stdunicode.IsSymbol},
		{"IsMark", unicode.IsMark, stdunicode.IsMark},
		{"IsControl", unicode.IsControl, stdunicode.IsControl},
		{"IsPrint", unicode.IsPrint, stdunicode.IsPrint},
		{"IsGraphic", unicode.IsGraphic, stdunicode.IsGraphic},
		{"IsSpace", unicode.IsSpace, stdunicode.IsSpace},
	}
	for r := rune(-1); r <= stdunicode.MaxRune+1; r++ {
		for _, p := range preds {
			if p.got(r) != p.want(r) {
				t.Fatalf("%s(%U) = %v", p.name, r, p.got(r))
			}
		}
		if unicode.ToUpper(r) != stdunicode.ToUpper(r) || unicode.ToLower(r) != stdunicode.ToLower(r) || unicode.ToTitle(r) != stdunicode.ToTitle(r) {
			t.Fatalf("case mapping of %U", r)
		}
		if unicode.SimpleFold(r) != stdunicode.SimpleFold(r) {
			t.Fatalf("SimpleFold(%U) = %U want %U", r, unicode.SimpleFold(r), stdunicode.SimpleFold(r))
		}
	}
}
