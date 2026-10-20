package language

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/testutil"
)

// Keep the shipped launcher covered alongside CI's shared runtime objects.
func TestSwiftStandaloneBuild(t *testing.T) {
	if only := os.Getenv("GOALCHEMY_TEST_TARGETS"); only == "" || !strings.Contains(","+only+",", ",swift,") {
		t.Skip("Swift suite only")
	}
	if os.Getenv("GOALCHEMY_TEST_BUILD_CACHE") == "" {
		t.Setenv("GOALCHEMY_TEST_BUILD_CACHE", t.TempDir())
	}
	fixture := testutil.Fixture{Dir: "testdata/co_gc_await", Gate: "cooperative"}
	res, ds := testutil.NewSuite(t).Build(fixture)
	if len(ds) > 0 {
		t.Fatal(ds)
	}
	standalone, cached := t.TempDir(), t.TempDir()
	for _, dir := range []string{standalone, cached} {
		if ds := driver.EmitWithOptions("swift", res, dir, driver.EmitOptions{}); len(ds) > 0 {
			t.Fatal(ds)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "run.sh")
	cmd.Dir, cmd.Env = standalone, driver.ToolEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("standalone Swift: %v\n%s", err, stderr.String())
	}
	want := testutil.Observation{Stdout: stdout.String(), Stderr: stderr.String()}
	got, err := testutil.Runners["swift"](cached)
	if err != nil || got != want {
		t.Fatalf("shared objects differ from standalone launcher: %v\nwant %s\ngot %s", err, want, got)
	}
}
