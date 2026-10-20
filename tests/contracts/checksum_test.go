package contracts

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/catalog"
	"github.com/eugenioenko/goalchemy/internal/driver"
)

// Canonical checksum cases run in TestTargetConformance with every other capability.
func TestChecksumCapabilities(t *testing.T) {
	cat, ds := catalog.Load(os.DirFS(root))
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	for _, name := range []string{"go", "typescript", "python", "java", "csharp", "rust", "c", "swift"} {
		t.Run(name, func(t *testing.T) {
			impl, ok := cat.Targets[name].Function("lib.checksum.crc32_ieee")
			if !ok || impl.Harness == "" {
				t.Fatal("missing CRC harness capability")
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
