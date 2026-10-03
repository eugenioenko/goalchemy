package driver

import (
	"github.com/eugenioenko/goalchemy"
	"github.com/eugenioenko/goalchemy/internal/catalog"
	"testing"
)

func TestCProductionCapabilitiesAreMapped(t *testing.T) {
	cat, ds := catalog.Load(goalchemy.Assets)
	if len(ds) != 0 {
		t.Fatal(ds)
	}
	for _, id := range []string{"lib.crypto.generate_p256", "lib.crypto.close", "lib.crypto.hmac_sha256_verify", "lib.http.do", "lib.callback.request", "lib.encoding.base64_decode"} {
		if cat.Functions[id] == nil {
			t.Fatal(id)
		}
		if _, ok := cat.Targets["c"].Function(id); !ok {
			t.Fatal("unavailable C capability", id)
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
