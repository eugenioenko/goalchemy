package integration

import (
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/eugenioenko/goalchemy/internal/project"
	"github.com/eugenioenko/goalchemy/internal/testutil"
)

// TestExamples builds each example from its goalchemy.yaml and compares
// every target with native Go.
//
// An example whose main.go carries "// goalchemy:network" fetches data over
// the network. It is skipped when GOALCHEMY_OFFLINE=1 or when
// raw.githubusercontent.com:443 cannot be reached within a few seconds.
func TestExamples(t *testing.T) {
	dirs, _ := filepath.Glob("../../examples/*")
	if len(dirs) == 0 {
		t.Fatal("no examples")
	}
	for _, dir := range dirs {
		cfg, err := project.Load(filepath.Join(dir, project.FileName))
		if err != nil {
			t.Fatal(err)
		}
		f := testutil.Fixture{Name: filepath.Base(dir), Dir: dir, Gate: cfg.Gate}
		src, _ := os.ReadFile(filepath.Join(dir, "main.go"))
		f.Unordered = testutil.HasDirective(string(src), "unordered")
		network := testutil.HasDirective(string(src), "network")
		t.Run(f.Name, func(t *testing.T) {
			if network {
				if reason := offline(); reason != "" {
					t.Skip("network example: " + reason)
				}
			}
			t.Parallel()
			targets := cfg.TargetNames()
			if testing.Short() {
				targets = []string{"go", "typescript"}
			}
			testutil.RunFixture(t, f, targets)
		})
	}
}

var offline = sync.OnceValue(func() string {
	if os.Getenv("GOALCHEMY_OFFLINE") == "1" {
		return "GOALCHEMY_OFFLINE=1"
	}
	conn, err := net.DialTimeout("tcp", "raw.githubusercontent.com:443", 5*time.Second)
	if err != nil {
		return "network unavailable: " + err.Error()
	}
	conn.Close()
	return ""
})
