package contracts

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/testutil"
)

func TestTypeScriptByteStorage(t *testing.T) {
	cmd := exec.Command("node", "--test", "targets/typescript/tests/byte_storage.test.ts")
	cmd.Dir = root
	cmd.Env = driver.ToolEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native byte storage: %v\n%s", err, out)
	}
	// Language behavior alone could pass with boxed arrays. Inspect the actual
	// compiler output as well as asserting the runtime's backing representation.
	outDir := t.TempDir()
	if ds := testutil.CompileGate("../language/testdata/byte_storage", "typescript", outDir, "sequential"); len(ds) > 0 {
		t.Fatal(ds)
	}
	src, err := testutil.GeneratedSource(outDir, ".ts")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"new Uint8Array(4)", "s: Uint8Array", "rt.BYTE_NIL", ", () => 0, true)"} {
		if !strings.Contains(string(src), want) {
			t.Errorf("emitted byte program lacks %q", want)
		}
	}
}

// Specialized backends opt into this stronger regression as they implement
// byte storage. The default bounded assignment verifies TypeScript against Go.
func TestTypeScriptByteGrowthNativeOracle(t *testing.T) {
	targets := []string{"typescript"}
	if only := os.Getenv("GOALCHEMY_BYTE_TARGETS"); only != "" {
		targets = strings.Split(only, ",")
	}
	testutil.RunFixture(t, testutil.Fixture{Name: "byte_growth", Dir: "testdata/byte_growth"}, targets)
}
