package contracts

import (
	"context"
	"github.com/eugenioenko/goalchemy/internal/driver"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestPythonNativeCapabilities(t *testing.T) {
	ctx, stop := context.WithTimeout(context.Background(), 120*time.Second)
	defer stop()
	cmd := exec.CommandContext(ctx, "python3", filepath.Join(root, "targets/python/tests/native_capabilities_test.py"))
	cmd.Env = driver.ToolEnv()
	if data, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("actual pinned native capability checks: %v\n%s", e, data)
	} else {
		t.Log(string(data))
	}
}
