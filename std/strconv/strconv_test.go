package strconv_test

import (
	"math"
	"math/rand"
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

func TestQuoteRuneMatchesStd(t *testing.T) {
	runes := []rune{0, 'a', '\'', '"', '\\', '\n', 0x7f, 0xa0, 0xad, 'é', '日', 0x1F600, 0xFEFF, 0xD800, 0x10FFFF, 0x110000, -1, utf8.RuneError}
	for _, r := range runes {
		if g, w := strconv.QuoteRune(r), stdstrconv.QuoteRune(r); g != w {
			t.Errorf("QuoteRune(%U) = %s want %s", r, g, w)
		}
		if g, w := strconv.QuoteRuneToASCII(r), stdstrconv.QuoteRuneToASCII(r); g != w {
			t.Errorf("QuoteRuneToASCII(%U) = %s want %s", r, g, w)
		}
	}
	for _, s := range []string{"", "abc", "a\tb", "a\nb", "a`b", "\x7f", "\xff", "é", "\ufeff", "日本"} {
		if g, w := strconv.CanBackquote(s), stdstrconv.CanBackquote(s); g != w {
			t.Errorf("CanBackquote(%q) = %v want %v", s, g, w)
		}
	}
	if g, w := string(strconv.AppendQuote([]byte("x"), "y\n")), string(stdstrconv.AppendQuote([]byte("x"), "y\n")); g != w {
		t.Errorf("AppendQuote = %s want %s", g, w)
	}
}

var floatInputs = []float64{
	0, 1, -1, 0.1, 0.2, 0.3, 1.0 / 3, 2.0 / 3, 100, 1e6, 1e20, 1e21, 1e-4, 1e-5, 1e-7, 123456789, 1234567890123456789,
	0.000001234, 5e-324, 1e-323, 2.2250738585072014e-308, 2.225073858507201e-308, 1.7976931348623157e308,
	3.4028234663852886e38, 1.401298464324817e-45, 1.1754943508222875e-38, 0.5, 1.5, 2.5, 3.5, 0.125, 1e23, 8.41e21,
	5.0e-1, 4.35, 0.05, 0.15, 0.25, 0.35, 123.456, 9.999999999999999e22, math.Pi, math.E, 299792458, 6.02214076e23,
	math.Inf(1), math.Inf(-1), math.NaN(), math.Copysign(0, -1), -1.5e-300, 1 << 53, 1<<53 + 1, 1 << 63, 4503599627370496.5,
}

func TestFormatFloatMatchesStd(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	inputs := append([]float64(nil), floatInputs...)
	for i := 0; i < 3000; i++ {
		inputs = append(inputs, math.Float64frombits(r.Uint64()), float64(math.Float32frombits(r.Uint32())),
			float64(r.Int63n(1000000))/1000, r.NormFloat64()*math.Pow(10, float64(r.Intn(40)-20)))
	}
	for _, f := range inputs {
		for _, fmt := range []byte{'e', 'E', 'f', 'g', 'G', 'b', 'x', 'X', 'v'} {
			for _, prec := range []int{-1, 0, 1, 2, 3, 5, 6, 10, 17, 20, 40} {
				for _, bits := range []int{32, 64} {
					g := strconv.FormatFloat(f, fmt, prec, bits)
					w := stdstrconv.FormatFloat(f, fmt, prec, bits)
					if g != w {
						t.Fatalf("FormatFloat(%v, %q, %d, %d) = %s want %s", f, fmt, prec, bits, g, w)
					}
				}
			}
		}
	}
	if g, w := string(strconv.AppendFloat([]byte("x"), 1.5, 'g', -1, 64)), "x1.5"; g != w {
		t.Errorf("AppendFloat = %s want %s", g, w)
	}
}

var parseFloatInputs = []string{
	"", "0", "-0", "+0", "1", "-1", "1.5", ".5", "5.", ".", "-.", "1e", "1e+", "1e-5", "1E5", "1e400", "-1e400", "1e-400",
	"4.9e-324", "2.4703282292062327e-324", "2.4703282292062328e-324", "2.2250738585072011e-308",
	"2.2250738585072012e-308", "1.7976931348623157e308", "1.7976931348623158e308", "1.7976931348623159e308",
	"179769313486231580793728971405301e276", "0.1", "0.2", "0.3", "123456789012345678901234567890",
	"9007199254740993", "9007199254740992.5", "1_000.5", "1__0", "_1", "0x1p-2", "0x1.8p1", "0X1P+3", "0x1", "0x.8p1",
	"0x1.fffffffffffff8p1023", "0x1p-1074", "0x1p-1075", "0x1.0000000000001p-1075", "inf", "-Inf", "+INF", "infinity",
	"-Infinity", "infin", "nan", "NaN", "-nan", "+nan", "nanx", "1x", "1.2.3", " 1", "1 ", "0.000000000000000000000000000001",
	"3.4028235e38", "3.4028236e38", "1.401298464324817e-45", "7e-46", "1e23", "8.98846567431158e307",
	"100000000000000016777215", "100000000000000016777216", "1.00000017881393432617187499", "1.000000178813934326171875",
	"6.9294956446009195e15", "0.500000000000000166533453693773481063544750213623046875",
}

func TestParseFloatMatchesStd(t *testing.T) {
	r := rand.New(rand.NewSource(2))
	inputs := append([]string(nil), parseFloatInputs...)
	for i := 0; i < 4000; i++ {
		f := math.Float64frombits(r.Uint64())
		inputs = append(inputs, stdstrconv.FormatFloat(f, 'g', -1, 64), stdstrconv.FormatFloat(f, 'e', r.Intn(25), 64),
			stdstrconv.FormatFloat(f, 'x', -1, 64), stdstrconv.FormatFloat(float64(math.Float32frombits(r.Uint32())), 'g', 17, 32))
	}
	for _, s := range inputs {
		for _, bits := range []int{32, 64} {
			g, gerr := strconv.ParseFloat(s, bits)
			w, werr := stdstrconv.ParseFloat(s, bits)
			if math.Float64bits(g) != math.Float64bits(w) && !(g != g && w != w) || errText(gerr) != errText(werr) {
				t.Fatalf("ParseFloat(%q, %d) = %v, %v want %v, %v", s, bits, g, gerr, w, werr)
			}
		}
	}
}
