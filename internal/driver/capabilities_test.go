package driver

import (
	"github.com/eugenioenko/goalchemy"
	"github.com/eugenioenko/goalchemy/internal/catalog"
	"github.com/eugenioenko/goalchemy/internal/ir"
	"testing"
)

func TestUnimplementedCapabilitiesFailBeforeEmission(t *testing.T) {
	cat, ds := catalog.Load(goalchemy.Assets)
	if len(ds) != 0 {
		t.Fatal(ds)
	}
	for _, target := range []string{"c"} {
		result := &Result{Catalog: cat, IR: &ir.Program{Externals: map[string]*ir.Extern{"probe": {Contract: "lib.crypto.generate_p256"}}}}
		ds := Emit(target, result, t.TempDir())
		if len(ds) != 1 || ds[0].Code != "GCE002" || ds[0].Symbol != "lib.crypto.generate_p256" {
			t.Fatalf("%s: %v", target, ds)
		}
	}
	for _, target := range []string{"c"} {
		result := &Result{Catalog: cat, IR: &ir.Program{Externals: map[string]*ir.Extern{"probe": {Contract: "lib.http.do"}}}}
		ds = Emit(target, result, t.TempDir())
		if len(ds) != 1 || ds[0].Code != "GCE002" || ds[0].Symbol != "lib.http.do" {
			t.Fatalf("%s: %v", target, ds)
		}
	}
}

func TestTypeOnlyOpaqueCapabilityRequiresImplementation(t *testing.T) {
	cat, ds := catalog.Load(goalchemy.Assets)
	if len(ds) != 0 {
		t.Fatal(ds)
	}
	result := &Result{Catalog: cat, IR: &ir.Program{Types: &ir.Types{All: []*ir.Type{{Kind: ir.KOpaque, Name: "crypto.Key", Obj: "Key", Pkg: "github.com/eugenioenko/goalchemy/lib/crypto"}}}}}
	for _, target := range []string{"c"} {
		ds := Emit(target, result, t.TempDir())
		if len(ds) != 1 || ds[0].Code != "GCE002" || ds[0].Symbol != "lib.crypto.close" {
			t.Fatalf("%s: %v", target, ds)
		}
	}
}

func TestRustProductionCapabilitiesAreMapped(t *testing.T) {
	cat, ds := catalog.Load(goalchemy.Assets)
	if len(ds) != 0 {
		t.Fatal(ds)
	}
	for _, id := range []string{"lib.crypto.generate_p256", "lib.crypto.close", "lib.crypto.hmac_sha256_verify", "lib.http.do", "lib.callback.request", "lib.encoding.base64_decode"} {
		fn := cat.Functions[id]
		if fn == nil {
			t.Fatal(id)
		}
		if _, ok := cat.Targets["rust"].Function(id); !ok {
			t.Fatal("unavailable Rust capability", id)
		}
	}
}
