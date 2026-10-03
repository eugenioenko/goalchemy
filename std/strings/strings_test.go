package strings_test

import (
	"math/rand"
	"reflect"
	stdstrings "strings"
	"testing"
	"unicode"

	"github.com/eugenioenko/goalchemy/std/strings"
)

var alphabet = []string{"a", "b", "A", "B", " ", "\t", "\n", "é", "É", "ß", "ſ", "K", "k", "K", "日", "\xff", "\xc3", "ab", " ", " ", "Σ", "σ", "ς"}

func samples() []string {
	rng := rand.New(rand.NewSource(7))
	out := []string{"", " ", "a", "aaa", "a,b,,c", "  hello  world  ", "\xff\xfe", "ÉtÉ"}
	for i := 0; i < 600; i++ {
		var b stdstrings.Builder
		for j := rng.Intn(7); j > 0; j-- {
			b.WriteString(alphabet[rng.Intn(len(alphabet))])
		}
		out = append(out, b.String())
	}
	return out
}

func eq(t *testing.T, name string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: got %#v want %#v", name, got, want)
	}
}

func TestMatchesStd(t *testing.T) {
	ss := samples()
	seps := []string{"", "a", "ab", " ", "é", "\xff", "aa", ","}
	for _, s := range ss {
		eq(t, "Fields "+s, strings.Fields(s), stdstrings.Fields(s))
		eq(t, "TrimSpace", strings.TrimSpace(s), stdstrings.TrimSpace(s))
		eq(t, "ToUpper "+s, strings.ToUpper(s), stdstrings.ToUpper(s))
		eq(t, "ToLower "+s, strings.ToLower(s), stdstrings.ToLower(s))
		eq(t, "ToTitle", strings.ToTitle(s), stdstrings.ToTitle(s))
		eq(t, "ToValidUTF8", strings.ToValidUTF8(s, "?"), stdstrings.ToValidUTF8(s, "?"))
		eq(t, "Repeat", strings.Repeat(s, 3), stdstrings.Repeat(s, 3))
		eq(t, "FieldsFunc", strings.FieldsFunc(s, unicode.IsUpper), stdstrings.FieldsFunc(s, unicode.IsUpper))
		eq(t, "IndexFunc", strings.IndexFunc(s, unicode.IsUpper), stdstrings.IndexFunc(s, unicode.IsUpper))
		eq(t, "LastIndexFunc", strings.LastIndexFunc(s, unicode.IsUpper), stdstrings.LastIndexFunc(s, unicode.IsUpper))
		eq(t, "TrimFunc", strings.TrimFunc(s, unicode.IsLower), stdstrings.TrimFunc(s, unicode.IsLower))
		drop := func(r rune) rune {
			if r == 'a' {
				return -1
			}
			return unicode.ToUpper(r)
		}
		eq(t, "Map", strings.Map(drop, s), stdstrings.Map(drop, s))
		for _, r := range []rune{'a', 'é', 0xFFFD, -1, 0x110000, '日', 0xD800} {
			eq(t, "IndexRune", strings.IndexRune(s, r), stdstrings.IndexRune(s, r))
			eq(t, "ContainsRune", strings.ContainsRune(s, r), stdstrings.ContainsRune(s, r))
		}
		for _, sep := range seps {
			eq(t, "Index", strings.Index(s, sep), stdstrings.Index(s, sep))
			eq(t, "LastIndex", strings.LastIndex(s, sep), stdstrings.LastIndex(s, sep))
			eq(t, "Contains", strings.Contains(s, sep), stdstrings.Contains(s, sep))
			eq(t, "ContainsAny", strings.ContainsAny(s, sep), stdstrings.ContainsAny(s, sep))
			eq(t, "IndexAny", strings.IndexAny(s, sep), stdstrings.IndexAny(s, sep))
			eq(t, "LastIndexAny", strings.LastIndexAny(s, sep), stdstrings.LastIndexAny(s, sep))
			eq(t, "Count", strings.Count(s, sep), stdstrings.Count(s, sep))
			eq(t, "HasPrefix", strings.HasPrefix(s, sep), stdstrings.HasPrefix(s, sep))
			eq(t, "HasSuffix", strings.HasSuffix(s, sep), stdstrings.HasSuffix(s, sep))
			eq(t, "Split "+s+"|"+sep, strings.Split(s, sep), stdstrings.Split(s, sep))
			eq(t, "SplitAfter", strings.SplitAfter(s, sep), stdstrings.SplitAfter(s, sep))
			for _, n := range []int{-1, 0, 1, 2, 5} {
				eq(t, "SplitN", strings.SplitN(s, sep, n), stdstrings.SplitN(s, sep, n))
				eq(t, "SplitAfterN", strings.SplitAfterN(s, sep, n), stdstrings.SplitAfterN(s, sep, n))
				eq(t, "Replace "+s+"|"+sep, strings.Replace(s, sep, "<>", n), stdstrings.Replace(s, sep, "<>", n))
			}
			eq(t, "ReplaceAll", strings.ReplaceAll(s, sep, "x"), stdstrings.ReplaceAll(s, sep, "x"))
			eq(t, "Trim", strings.Trim(s, sep), stdstrings.Trim(s, sep))
			eq(t, "TrimLeft", strings.TrimLeft(s, sep), stdstrings.TrimLeft(s, sep))
			eq(t, "TrimRight", strings.TrimRight(s, sep), stdstrings.TrimRight(s, sep))
			eq(t, "TrimPrefix", strings.TrimPrefix(s, sep), stdstrings.TrimPrefix(s, sep))
			eq(t, "TrimSuffix", strings.TrimSuffix(s, sep), stdstrings.TrimSuffix(s, sep))
			gb, ga, gf := strings.Cut(s, sep)
			wb, wa, wf := stdstrings.Cut(s, sep)
			eq(t, "Cut", []any{gb, ga, gf}, []any{wb, wa, wf})
			eq(t, "Compare", strings.Compare(s, sep), stdstrings.Compare(s, sep))
		}
		for _, u := range ss[:80] {
			eq(t, "EqualFold "+s+"|"+u, strings.EqualFold(s, u), stdstrings.EqualFold(s, u))
		}
		eq(t, "EqualFold upper", strings.EqualFold(s, stdstrings.ToUpper(s)), stdstrings.EqualFold(s, stdstrings.ToUpper(s)))
		eq(t, "Join", strings.Join(strings.Split(s, "a"), "--"), stdstrings.Join(stdstrings.Split(s, "a"), "--"))
		for _, pairs := range [][]string{{"a", "1", "b", "2"}, {"a", "1", "ab", "2"}, {"ab", "2", "a", "1"}, {"", "-"}, {"", "-", "a", "A"}, {"a", "A", "", "-"}, {"é", "e", "\xff", "?"}} {
			eq(t, "Replacer "+s, strings.NewReplacer(pairs...).Replace(s), stdstrings.NewReplacer(pairs...).Replace(s))
		}
	}
}

func TestBuilder(t *testing.T) {
	var b strings.Builder
	b.WriteString("hé")
	b.WriteByte('!')
	b.WriteRune('日')
	b.WriteRune(-1)
	b.Write([]byte("x"))
	b.Grow(100)
	if b.String() != "hé!日�x" || b.Len() != len("hé!日�x") || b.Cap() < 100 {
		t.Fatal(b.String())
	}
	b.Reset()
	if b.String() != "" {
		t.Fatal("reset")
	}
}
