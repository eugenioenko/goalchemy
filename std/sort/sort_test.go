package sort_test

import (
	"math/rand"
	"reflect"
	"slices"
	stdsort "sort"
	"testing"

	"github.com/eugenioenko/goalchemy/std/sort"
)

type pair struct{ k, v int }

type byKey []pair

func (p byKey) Len() int           { return len(p) }
func (p byKey) Less(i, j int) bool { return p[i].k < p[j].k }
func (p byKey) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func TestMatchesStdStable(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	for n := 0; n < 300; n++ {
		a := make([]pair, n)
		for i := range a {
			a[i] = pair{rng.Intn(n/3 + 1), i}
		}
		g := append([]pair(nil), a...)
		w := append([]pair(nil), a...)
		sort.Stable(byKey(g))
		stdsort.Stable(byKey(w))
		if !reflect.DeepEqual(g, w) {
			t.Fatalf("Stable n=%d", n)
		}
		g = append([]pair(nil), a...)
		sort.Sort(sort.Reverse(byKey(g)))
		if !sort.IsSorted(sort.Reverse(byKey(g))) {
			t.Fatalf("Reverse n=%d", n)
		}
		ints := make([]int, n)
		strs := make([]string, n)
		for i := range ints {
			ints[i] = rng.Intn(50) - 25
			strs[i] = string(rune('a' + rng.Intn(26)))
		}
		wi := append([]int(nil), ints...)
		ws := append([]string(nil), strs...)
		sort.Ints(ints)
		sort.Strings(strs)
		stdsort.Ints(wi)
		stdsort.Strings(ws)
		if !slices.Equal(ints, wi) || !slices.Equal(strs, ws) || !sort.IntsAreSorted(ints) || !sort.StringsAreSorted(strs) {
			t.Fatalf("Ints/Strings n=%d", n)
		}
		for _, x := range []int{-30, 0, 3, 30} {
			if sort.SearchInts(ints, x) != stdsort.SearchInts(wi, x) {
				t.Fatal("SearchInts")
			}
		}
		if sort.SearchStrings(strs, "m") != stdsort.SearchStrings(ws, "m") {
			t.Fatal("SearchStrings")
		}
	}
}
