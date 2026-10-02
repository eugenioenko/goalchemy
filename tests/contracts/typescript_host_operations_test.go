package contracts

import (
	"os/exec"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/driver"
)

func TestTypeScriptHostOperations(t *testing.T) {
	cmd := exec.Command("node", "targets/typescript/tests/host_operations.test.ts")
	cmd.Dir = root
	cmd.Env = driver.ToolEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("actual TypeScript host runtime: %v\n%s", err, out)
	}
}
