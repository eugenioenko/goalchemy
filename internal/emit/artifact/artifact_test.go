package artifact

import (
	"strings"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/ir"
)

func TestCanonicalRepresentationOwnership(t *testing.T) {
	underlying := &ir.Type{ID: 0, Kind: ir.KStruct}
	anonymous := &ir.Type{ID: 1, Kind: ir.KStruct}
	first := &ir.Type{ID: 2, Kind: ir.KNamed, Name: "first.Value", Pkg: "example.com/first/value", Underlying: underlying}
	second := &ir.Type{ID: 3, Kind: ir.KNamed, Name: "second.Value", Pkg: "example.com/second/value", Underlying: underlying}
	p := &ir.Program{Types: &ir.Types{All: []*ir.Type{underlying, anonymous, second, first}}}
	l, err := NewLayout(p)
	if err != nil {
		t.Fatal(err)
	}
	if l.TypeOwner(first) != first.Pkg || l.TypeOwner(second) != second.Pkg || l.TypeOwner(underlying) != "" {
		t.Fatal("lost declared type ownership")
	}
	if l.RepresentationOwner(first) != first.Pkg || l.RepresentationOwner(second) != first.Pkg || l.RepresentationOwner(underlying) != first.Pkg || l.RepresentationOwner(anonymous) != "" {
		t.Fatal("duplicated shared representation ownership")
	}
	if l.Stem(first.Pkg) == l.Stem(second.Pkg) {
		t.Fatal("same-basename packages collide")
	}
	long := strings.Repeat("source", 100) + "/" + strings.Repeat("value", 100)
	l, err = NewLayout(&ir.Program{Packages: []ir.Package{{Path: long}}})
	if err != nil || len(l.Stem(long)) > 48 {
		t.Fatalf("unbounded path: %v", err)
	}
}

func TestSourceOwnerMustBeIncluded(t *testing.T) {
	_, err := NewLayout(&ir.Program{Packages: []ir.Package{{Path: "example.com/root"}}, Funcs: []*ir.Func{{Pkg: "example.com/missing"}}})
	if err == nil {
		t.Fatal("missing owner silently created a source package")
	}
}

func TestNamedCapabilityTypesRemainCentral(t *testing.T) {
	typ := &ir.Type{Kind: ir.KNamed, Name: "time.Duration", Pkg: "example.com/lib/time", Underlying: &ir.Type{Kind: ir.KInt}}
	l, err := NewLayout(&ir.Program{Packages: []ir.Package{{Path: "example.com/root"}}, Types: &ir.Types{All: []*ir.Type{typ}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Packages) != 1 || l.TypeOwner(typ) != "" || l.RepresentationOwner(typ) != "" {
		t.Fatal("capability type became an included source package")
	}
}
