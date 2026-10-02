package contracts

import (
	"context"
	"github.com/eugenioenko/goalchemy/internal/driver"
	"os/exec"
	"testing"
	"time"
)

func TestRustHostOperations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "targets/rust/tests/host_operations_test.sh")
	cmd.Dir = root
	cmd.Env = driver.ToolEnv()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("actual Rust host lifecycle: %v\n%s", err, output)
	}
	t.Log(string(output))
}
