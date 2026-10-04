package naming

import (
	"regexp"
	"strings"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/ir"
)

func TestCanonicalTypeAndCompactNames(t *testing.T) {
	u := &ir.Type{ID: 3, Kind: ir.KStruct}
	z := &ir.Type{ID: 4, Kind: ir.KNamed, Name: "z.Widget", Underlying: u}
	a := &ir.Type{ID: 5, Kind: ir.KNamed, Name: "a.Widget", Underlying: u}
	p := &ir.Program{Types: &ir.Types{All: []*ir.Type{u, z, a}}}
	for _, all := range [][]*ir.Type{{u, z, a}, {a, z, u}} {
		p.Types.All = all
		if got := New(p).Type(u, "S_"); got != "S_a_Widget_3" {
			t.Fatalf("canonical type = %q", got)
		}
		if got := Type(p, z, "S_"); got != "S_z_Widget_4" {
			t.Fatalf("named type = %q", got)
		}
	}
	p.CompactNames = true
	if got := New(p).Type(u, "S_"); got != "S_3" {
		t.Fatalf("compact type = %q", got)
	}
}

func TestSafeSourceNamesAndShadowing(t *testing.T) {
	p := &ir.Program{}
	valid := regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	seen := map[string]bool{}
	for id, name := range []string{"class", "λ", "a-b", "a_b", "λ", "", "_"} {
		l := &ir.Local{ID: id, Name: name, Kind: ir.LTemp}
		s := Local(p, l, "_")
		if !valid.MatchString(s) || seen[s] {
			t.Fatalf("unsafe or duplicate local %q", s)
		}
		seen[s] = true
	}
	if got := Local(p, &ir.Local{ID: 8, Kind: ir.LParam}, "_"); got != "v_param_8" {
		t.Fatalf("unnamed param = %q", got)
	}
	if got := Local(p, &ir.Local{ID: 9, Kind: ir.LResult}, "_"); got != "v_result_9" {
		t.Fatalf("unnamed result = %q", got)
	}
	if got := Identifier("λ"); got != "_u3bb_" {
		t.Fatalf("unicode encoding = %q", got)
	}
}

func TestPrivateSymbolsAndFields(t *testing.T) {
	f := &ir.Func{ID: 2, Sym: "main_Do", Name: "main.Do"}
	g := &ir.Global{ID: 2, Sym: "main_Count", Name: "Count", Pkg: "main"}
	u := &ir.Type{Kind: ir.KStruct, Fields: []ir.Field{{Name: "class"}, {Name: "λ"}}}
	p := &ir.Program{Funcs: []*ir.Func{f}, Globals: []*ir.Global{g}}
	readable := New(p)
	copy := *p
	copy.CompactNames = true
	short := New(&copy)
	for _, tc := range []struct{ symbol, readable, compact string }{
		{f.Sym, "fn_main_Do_2", "f2"},
		{g.Sym, "global_main_Count_2", "g2"},
		{"rt.external", "rt.external", "rt.external"},
	} {
		if got := readable.Symbol(tc.symbol); got != tc.readable {
			t.Fatalf("readable symbol = %q", got)
		}
		if got := short.Symbol(tc.symbol); got != tc.compact {
			t.Fatalf("compact symbol = %q", got)
		}
		if got := Symbol(p, tc.symbol); got != tc.readable {
			t.Fatalf("helper symbol = %q", got)
		}
	}
	if got := readable.Field(u, 0, "f_"); got != "f_class_0" {
		t.Fatalf("readable field = %q", got)
	}
	if got := short.Field(u, 0, "f_"); got != "f_0" {
		t.Fatalf("compact field = %q", got)
	}
	if !strings.Contains(readable.Field(u, 1, "f_"), "u3bb") {
		t.Fatal("lost Unicode field identity")
	}
	if p.CompactNames || f.Sym != "main_Do" || g.Sym != "main_Count" || u.Fields[0].Name != "class" {
		t.Fatal("naming modified source identities")
	}
	unicodeGlobal := &ir.Global{ID: 3, Sym: "main__", Name: "λ", Pkg: "example.com/main"}
	p.Globals = append(p.Globals, unicodeGlobal)
	if got := New(p).Symbol(unicodeGlobal.Sym); !strings.Contains(got, "_u3bb_") {
		t.Fatalf("lost Unicode global identity: %q", got)
	}
}
