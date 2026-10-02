package contracts

import (
	"context"
	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/testutil"
	"os/exec"
	"testing"
	"time"
)

func TestCHostOperations(t *testing.T) {
	cHostRuntime(t, nil)
}
func TestCHostOperationsSanitized(t *testing.T) {
	clang, err := testutil.SanitizerClang()
	if err != nil {
		t.Fatal(err)
	}
	cHostRuntime(t, []string{"CC=" + clang, "CFLAGS=-O1 -g -fsanitize=address,undefined -fno-sanitize-recover=all", "ASAN_OPTIONS=detect_stack_use_after_return=0:detect_leaks=0"})
}
func cHostRuntime(t *testing.T, env []string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "targets/c/tests/host_operations_test.sh")
	cmd.Dir = root
	cmd.Env = append(driver.ToolEnv(), env...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("actual C lifecycle: %v\n%s", err, b)
	}
	t.Log(string(b))
}
