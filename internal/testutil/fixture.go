// Package testutil runs language fixtures natively and through Goalchemy
// targets and compares their observations.
package testutil

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"goalchemy/internal/diagnostics"
	"goalchemy/internal/driver"
	"goalchemy/internal/frontend"
)

// Observation is what a fixture run produced.
type Observation struct {
	Stdout string
	Stderr string
	Exit   int
}

func (o Observation) String() string {
	return fmt.Sprintf("exit=%d\n--- stdout\n%s--- stderr\n%s", o.Exit, o.Stdout, o.Stderr)
}

var panicTail = regexp.MustCompile(`(?s)\n(\[signal [^\n]*\]\n)?\ngoroutine \d+ .*$`)

// Normalize keeps standard error up to and including the first panic line
// and drops goroutine traces, which are not part of the observable contract.
func Normalize(o Observation) Observation {
	if i := strings.Index(o.Stderr, "panic: "); i >= 0 && (i == 0 || o.Stderr[i-1] == '\n') {
		rest := o.Stderr[i:]
		if j := strings.IndexByte(rest, '\n'); j >= 0 {
			rest = rest[:j+1]
		}
		o.Stderr = o.Stderr[:i] + strings.TrimSuffix(rest, " [recovered]\n")
		if !strings.HasSuffix(o.Stderr, "\n") {
			o.Stderr += "\n"
		}
	}
	o.Stderr = panicTail.ReplaceAllString(o.Stderr, "\n")
	return o
}

// Unordered sorts output lines, for fixtures whose native order is unspecified.
func Unordered(o Observation) Observation {
	sortLines := func(s string) string {
		lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
		sort.Strings(lines)
		return strings.Join(lines, "\n") + "\n"
	}
	o.Stdout = sortLines(o.Stdout)
	o.Stderr = sortLines(o.Stderr)
	return o
}

func run(dir string, timeout time.Duration, name string, args ...string) (Observation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN="+frontend.ReferenceToolchain, "GOFLAGS=")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	o := Observation{Stdout: out.String(), Stderr: errb.String()}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		o.Exit = ee.ExitCode()
		err = nil
	}
	if ctx.Err() != nil {
		return o, fmt.Errorf("%s timed out", name)
	}
	return o, err
}

var buildMu sync.Mutex

// Native builds and runs the fixture with the reference Go toolchain.
func Native(fixtureDir, work string) (Observation, error) {
	bin := filepath.Join(work, "native")
	abs, _ := filepath.Abs(fixtureDir)
	if o, err := run(abs, 2*time.Minute, "go", "build", "-o", bin, "."); err != nil || o.Exit != 0 {
		return o, fmt.Errorf("native build failed: %v\n%s", err, o.Stderr)
	}
	return run(work, time.Minute, bin)
}

// Compile compiles the fixture for target into out.
func Compile(fixtureDir, target, out string) []diagnostics.Diagnostic {
	abs, _ := filepath.Abs(fixtureDir)
	res, ds := driver.Build(context.Background(), driver.Options{Dir: abs})
	if diagnostics.HasErrors(ds) {
		return ds
	}
	return driver.Emit(target, res, out)
}

// Runner knows how to build and run a compiled target directory.
type Runner func(out string) (Observation, error)

var Runners = map[string]Runner{
	"go": func(out string) (Observation, error) {
		if o, err := run(out, 2*time.Minute, "go", "build", "-o", "prog", "."); err != nil || o.Exit != 0 {
			return o, fmt.Errorf("go build failed: %v\n%s", err, o.Stderr)
		}
		return run(out, time.Minute, filepath.Join(out, "prog"))
	},
}

// Fixture describes one language fixture directory.
type Fixture struct {
	Name string
	Dir  string
	// Unordered marks fixtures whose output order differs from native Go
	// only by unspecified map iteration order.
	Unordered bool
	// Reject lists diagnostic codes the fixture must produce.
	Reject []string
}

var directive = regexp.MustCompile(`(?m)^// goalchemy:(\w+)(?: (.*))?$`)

func Discover(root string) ([]Fixture, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []Fixture
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		f := Fixture{Name: e.Name(), Dir: filepath.Join(root, e.Name())}
		src, _ := os.ReadFile(filepath.Join(f.Dir, "main.go"))
		for _, m := range directive.FindAllStringSubmatch(string(src), -1) {
			switch m[1] {
			case "unordered":
				f.Unordered = true
			case "reject":
				f.Reject = strings.Fields(m[2])
			}
		}
		out = append(out, f)
	}
	return out, nil
}
