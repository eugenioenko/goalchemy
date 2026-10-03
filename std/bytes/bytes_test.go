package bytes_test

import (
	stdbytes "bytes"
	"math/rand"
	"reflect"
	"testing"

	"github.com/eugenioenko/goalchemy/std/bytes"
)

func samples() [][]byte {
	parts := []string{"a", "b", " ", "\t", "é", "\xff", "ab", ",", " "}
	rng := rand.New(rand.NewSource(3))
	out := [][]byte{nil, {}, []byte(" x  y "), []byte("a,b,,c")}
	for i := 0; i < 400; i++ {
		var b []byte
		for j := rng.Intn(6); j > 0; j-- {
			b = append(b, parts[rng.Intn(len(parts))]...)
		}
		out = append(out, b)
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
	for _, s := range ss {
		eq(t, "TrimSpace", bytes.TrimSpace(s), stdbytes.TrimSpace(s))
		eq(t, "Fields", bytes.Fields(s), stdbytes.Fields(s))
		eq(t, "ToUpper", bytes.ToUpper(s), stdbytes.ToUpper(s))
		eq(t, "ToLower", bytes.ToLower(s), stdbytes.ToLower(s))
		eq(t, "Repeat", bytes.Repeat(s, 2), stdbytes.Repeat(s, 2))
		eq(t, "Clone", bytes.Clone(s), stdbytes.Clone(s))
		for _, sep := range [][]byte{nil, []byte("a"), []byte(","), []byte("ab"), []byte("\xff")} {
			eq(t, "Equal", bytes.Equal(s, sep), stdbytes.Equal(s, sep))
			eq(t, "Compare", bytes.Compare(s, sep), stdbytes.Compare(s, sep))
			eq(t, "Index", bytes.Index(s, sep), stdbytes.Index(s, sep))
			eq(t, "LastIndex", bytes.LastIndex(s, sep), stdbytes.LastIndex(s, sep))
			eq(t, "Count", bytes.Count(s, sep), stdbytes.Count(s, sep))
			eq(t, "Contains", bytes.Contains(s, sep), stdbytes.Contains(s, sep))
			eq(t, "HasPrefix", bytes.HasPrefix(s, sep), stdbytes.HasPrefix(s, sep))
			eq(t, "HasSuffix", bytes.HasSuffix(s, sep), stdbytes.HasSuffix(s, sep))
			eq(t, "TrimPrefix", bytes.TrimPrefix(s, sep), stdbytes.TrimPrefix(s, sep))
			eq(t, "TrimSuffix", bytes.TrimSuffix(s, sep), stdbytes.TrimSuffix(s, sep))
			eq(t, "Split", bytes.Split(s, sep), stdbytes.Split(s, sep))
			for _, n := range []int{-1, 0, 1, 2} {
				eq(t, "SplitN", bytes.SplitN(s, sep, n), stdbytes.SplitN(s, sep, n))
			}
			eq(t, "Join", bytes.Join(bytes.Split(s, sep), []byte("-")), stdbytes.Join(stdbytes.Split(s, sep), []byte("-")))
			eq(t, "EqualFold", bytes.EqualFold(s, sep), stdbytes.EqualFold(s, sep))
		}
		eq(t, "IndexByte", bytes.IndexByte(s, 'a'), stdbytes.IndexByte(s, 'a'))
		eq(t, "IndexRune", bytes.IndexRune(s, 'é'), stdbytes.IndexRune(s, 'é'))
	}
	eq(t, "Join empty", bytes.Join(nil, []byte(",")), stdbytes.Join(nil, []byte(",")))
}

func TestSplitAliases(t *testing.T) {
	s := []byte("a,b")
	parts := bytes.Split(s, []byte(","))
	parts[0][0] = 'X'
	if s[0] != 'X' || cap(parts[0]) != 1 {
		t.Fatal("Split must return capacity-limited subslices")
	}
}

func TestBuffer(t *testing.T) {
	var g bytes.Buffer
	var w stdbytes.Buffer
	for _, s := range []string{"hé", "", "llo", "\xff"} {
		g.WriteString(s)
		w.WriteString(s)
	}
	g.WriteByte('!')
	w.WriteByte('!')
	g.WriteRune('日')
	w.WriteRune('日')
	eq(t, "Next", g.Next(2), w.Next(2))
	gc, ge := g.ReadByte()
	wc, we := w.ReadByte()
	eq(t, "ReadByte", []any{gc, ge == nil}, []any{wc, we == nil})
	eq(t, "String", g.String(), w.String())
	g.Truncate(2)
	w.Truncate(2)
	eq(t, "Truncate", g.Bytes(), w.Bytes())
	eq(t, "Len", g.Len(), w.Len())
	var nb *bytes.Buffer
	eq(t, "nil String", nb.String(), "<nil>")
}
