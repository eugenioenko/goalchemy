package strconv_test

import (
	"math"
	stdstrconv "strconv"
	"testing"
	"unicode/utf8"

	"github.com/eugenioenko/goalchemy/std/strconv"
)

var intInputs = []string{
	"", "0", "-0", "+0", "1", "-1", "+1", "42", "007", "0x1F", "0X1f", "0b101", "0o17", "017", "0_1", "1_000",
	"1__0", "_1", "1_", "0x_1f", "0x", "0b", "-0x80", "9223372036854775807", "9223372036854775808",
	"-9223372036854775808", "-9223372036854775809", "18446744073709551615", "18446744073709551616",
	"99999999999999999999999", "zz", "Z", "12a", " 1", "1 ", "--1", "+-1", "-", "+", "0xffffffffffffffff", "127", "128", "-129", "255", "256",
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestParseMatchesStd(t *testing.T) {
	for _, s := range intInputs {
		for _, base := range []int{0, 2, 8, 10, 16, 36, 1, 37} {
			for _, bits := range []int{0, 8, 16, 32, 64, 65} {
				gi, ge := strconv.ParseInt(s, base, bits)
				wi, we := stdstrconv.ParseInt(s, base, bits)
				if gi != wi || errText(ge) != errText(we) {
					t.Errorf("ParseInt(%q,%d,%d) = %d,%v want %d,%v", s, base, bits, gi, ge, wi, we)
				}
				gu, ge := strconv.ParseUint(s, base, bits)
				wu, we := stdstrconv.ParseUint(s, base, bits)
				if gu != wu || errText(ge) != errText(we) {
					t.Errorf("ParseUint(%q,%d,%d) = %d,%v want %d,%v", s, base, bits, gu, ge, wu, we)
				}
			}
		}
		ga, ge := strconv.Atoi(s)
		wa, we := stdstrconv.Atoi(s)
		if ga != wa || errText(ge) != errText(we) {
			t.Errorf("Atoi(%q) = %d,%v want %d,%v", s, ga, ge, wa, we)
		}
		gb, ge := strconv.ParseBool(s)
		wb, we := stdstrconv.ParseBool(s)
		if gb != wb || errText(ge) != errText(we) {
			t.Errorf("ParseBool(%q) = %v,%v want %v,%v", s, gb, ge, wb, we)
		}
	}
	for _, s := range []string{"true", "True", "TRUE", "t", "T", "1", "false", "f", "0", "yes"} {
		gb, ge := strconv.ParseBool(s)
		wb, we := stdstrconv.ParseBool(s)
		if gb != wb || errText(ge) != errText(we) {
			t.Errorf("ParseBool(%q)", s)
		}
	}
}

func TestFormatMatchesStd(t *testing.T) {
	ints := []int64{0, 1, -1, 7, -8, 255, 1 << 31, math.MaxInt64, math.MinInt64, -1234567890}
	for _, i := range ints {
		for base := 2; base <= 36; base++ {
			if g, w := strconv.FormatInt(i, base), stdstrconv.FormatInt(i, base); g != w {
				t.Errorf("FormatInt(%d,%d) = %q want %q", i, base, g, w)
			}
			if g, w := strconv.FormatUint(uint64(i), base), stdstrconv.FormatUint(uint64(i), base); g != w {
				t.Errorf("FormatUint(%d,%d) = %q want %q", uint64(i), base, g, w)
			}
		}
		if g, w := string(strconv.AppendInt([]byte("x"), i, 10)), string(stdstrconv.AppendInt([]byte("x"), i, 10)); g != w {
			t.Errorf("AppendInt %q %q", g, w)
		}
	}
	if strconv.Itoa(-42) != "-42" || strconv.FormatBool(true) != "true" || strconv.FormatBool(false) != "false" {
		t.Error("Itoa/FormatBool")
	}
}

func TestQuoteMatchesStd(t *testing.T) {
	for _, s := range []string{"", "abc", "a\"b\\c", "\a\b\f\n\r\t\v\x00\x1f\x7f", "\xff\xfe", "é ü 日本 😀", "\u00ad\u200b\u2028\ufeff", "\xc3", "\xed\xa0\x80"} {
		if g, w := strconv.Quote(s), stdstrconv.Quote(s); g != w {
			t.Errorf("Quote(%q) = %s want %s", s, g, w)
		}
		if g, w := strconv.QuoteToASCII(s), stdstrconv.QuoteToASCII(s); g != w {
			t.Errorf("QuoteToASCII(%q) = %s want %s", s, g, w)
		}
	}
}

func TestQuoteAssignedRunes(t *testing.T) {
	var bad int
	var first rune = -1
	for r := rune(0); r <= utf8.MaxRune; r++ {
		if r >= 0xd800 && r <= 0xdfff {
			continue
		}
		s := string(r)
		if strconv.Quote(s) != stdstrconv.Quote(s) {
			bad++
			if first < 0 {
				first = r
			}
		}
	}
	if bad > 0 {
		t.Errorf("Quote differs from std on %d code points (first %U)", bad, first)
	}
}
