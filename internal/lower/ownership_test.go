package lower_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/diagnostics"
	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/subset"
)

// Native Go lvalues can conceal missing address metadata. Backends that use
// storage cells must agree on qualified and unqualified source globals.
func TestQualifiedGlobalAddressAndInitializerClosureOwnership(t *testing.T) {
	for _, fixture := range []string{"../../tests/language/testdata/package_output", "../../tests/integration/testdata/package_library/api"} {
		path, err := filepath.Abs(fixture)
		if err != nil {
			t.Fatal(err)
		}
		res, ds := driver.Build(context.Background(), driver.Options{Dir: path, Gate: subset.Cooperative})
		if diagnostics.HasErrors(ds) {
			t.Fatal(ds)
		}
		if strings.Contains(fixture, "package_output") {
			found := false
			for _, global := range res.IR.Globals {
				if strings.HasSuffix(global.Pkg, "/first/value") && global.Name == "Counter" {
					found = true
					if !global.AddrTaken {
						t.Fatal("qualified scalar global address lacks stable storage metadata")
					}
				}
			}
			if !found {
				t.Fatal("missing cross-package pointer global")
			}
		} else {
			found := false
			for _, function := range res.IR.Funcs {
				if function.Closure && strings.HasPrefix(function.Name, "$init$") {
					found = true
					if !strings.HasSuffix(function.Pkg, "/state") {
						t.Fatalf("initializer closure is owned by %q", function.Pkg)
					}
				}
			}
			if !found {
				t.Fatal("missing initializer closure")
			}
		}
	}
}
