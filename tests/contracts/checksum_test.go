package contracts

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eugenioenko/goalchemy/internal/catalog"
	"github.com/eugenioenko/goalchemy/internal/conformance"
	"github.com/eugenioenko/goalchemy/internal/contracts"
	"github.com/eugenioenko/goalchemy/internal/driver"
)

// Exercise only the new capability cases through all seven actual host harnesses.
func TestChecksumTargetConformance(t *testing.T) {
	for _, kv := range driver.ToolEnv() {
		k, v, _ := strings.Cut(kv, "=")
		if os.Getenv(k) != v {
			t.Setenv(k, v)
		}
	}
	cat, ds := catalog.Load(os.DirFS(root))
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	const id = "lib.checksum.crc32_ieee"
	for _, name := range []string{"go", "typescript", "python", "java", "csharp", "rust", "c"} {
		t.Run(name, func(t *testing.T) {
			target := cat.Targets[name]
			impl, ok := target.Function(id)
			if !ok {
				t.Fatal("missing CRC capability")
			}
			target.Functions = []contracts.Implementation{impl}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			results, err := conformance.RunContext(ctx, cat, root, name)
			if err != nil {
				t.Fatal(err)
			}
			if len(results) != len(cat.Functions[id].Cases) {
				t.Fatalf("got %d cases", len(results))
			}
			for _, r := range results {
				if !r.Pass {
					t.Errorf("%s: %s", r.Case, r.Detail)
				}
			}
		})
	}
}

func TestTypeScriptChecksumPrimitive(t *testing.T) {
	cmd := exec.Command("node", "--test", "targets/typescript/tests/checksum.test.ts")
	cmd.Dir, cmd.Env = root, driver.ToolEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("CRC primitive: %v\n%s", err, out)
	}
}

func csharpChecksumDLL(t *testing.T, ctx context.Context) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, "sh", filepath.Join(root, "targets/csharp/tests/checksum-dependencies.sh"))
	cmd.Env = driver.ToolEnv()
	data, err := cmd.Output()
	if err != nil {
		t.Fatalf("locked C# checksum dependency: %v", err)
	}
	return strings.TrimSpace(string(data))
}
