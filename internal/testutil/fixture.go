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
	"testing"
	"time"

	"goalchemy/internal/diagnostics"
	"goalchemy/internal/driver"
	"goalchemy/internal/frontend"
	"goalchemy/internal/subset"
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

var fatalLine = regexp.MustCompile(`(?m)^fatal error: [^\n]*\n`)

var panicTail = regexp.MustCompile(`(?s)\n(\[signal [^\n]*\]\n)?\ngoroutine \d+ .*$`)

// Normalize keeps standard error up to and including the first panic line
// and drops goroutine traces, which are not part of the observable contract.
func Normalize(o Observation) Observation {
	if loc := fatalLine.FindStringIndex(o.Stderr); loc != nil {
		o.Stderr = o.Stderr[:loc[1]]
		return o
	}
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
	return CompileGate(fixtureDir, target, out, "")
}

// CompileGate compiles with an explicit language gate.
func CompileGate(fixtureDir, target, out, gate string) []diagnostics.Diagnostic {
	abs, _ := filepath.Abs(fixtureDir)
	res, ds := driver.Build(context.Background(), driver.Options{Dir: abs, Gate: subset.Gate(gate)})
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
	"typescript": func(out string) (Observation, error) {
		return run(out, time.Minute, "node", "--stack-size=4000", "main.ts")
	},
	"python": func(out string) (Observation, error) {
		return run(out, 2*time.Minute, "python3", "main.py")
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
	// Golden holds the expected normalized observation for fixtures that
	// assert Goalchemy-selected behavior native Go does not share.
	Golden string
	// Gate is the language gate the fixture needs.
	Gate string
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
		if want, err := os.ReadFile(filepath.Join(f.Dir, "want.txt")); err == nil {
			f.Golden = string(want)
		}
		for _, m := range directive.FindAllStringSubmatch(string(src), -1) {
			switch m[1] {
			case "unordered":
				f.Unordered = true
			case "reject":
				f.Reject = strings.Fields(m[2])
			case "gate":
				f.Gate = strings.TrimSpace(m[2])
			}
		}
		out = append(out, f)
	}
	return out, nil
}

// ParseObservation parses the String form of an Observation.
func ParseObservation(s string) Observation {
	var o Observation
	head, rest, _ := strings.Cut(s, "\n--- stdout\n")
	fmt.Sscanf(head, "exit=%d", &o.Exit)
	o.Stdout, o.Stderr, _ = strings.Cut(rest, "--- stderr\n")
	return o
}

// RunFixture runs one fixture through targets and reports mismatches.
func RunFixture(t *testing.T, f Fixture, targets []string) {
	t.Helper()
	work := t.TempDir()
	if len(f.Reject) > 0 {
		ds := CompileGate(f.Dir, targets[0], filepath.Join(work, targets[0]), f.Gate)
		got := map[string]bool{}
		for _, d := range ds {
			got[d.Code] = true
		}
		for _, c := range f.Reject {
			if !got[c] {
				t.Errorf("expected diagnostic %s, got %v", c, ds)
			}
		}
		return
	}
	var want Observation
	if f.Golden != "" {
		want = ParseObservation(f.Golden)
	} else {
		native, err := Native(f.Dir, work)
		if err != nil {
			t.Fatal(err)
		}
		want = Normalize(native)
	}
	var first *Observation
	for _, target := range targets {
		out := filepath.Join(work, target)
		if ds := CompileGate(f.Dir, target, out, f.Gate); len(ds) > 0 {
			for _, d := range ds {
				t.Errorf("%s: %s", target, d)
			}
			continue
		}
		obs, err := Runners[target](out)
		if err != nil {
			t.Errorf("%s: %v", target, err)
			continue
		}
		got := Normalize(obs)
		cmpWant, cmpGot := want, got
		if f.Unordered {
			cmpWant, cmpGot = Unordered(want), Unordered(got)
		}
		if cmpGot != cmpWant {
			t.Errorf("%s differs from expected\n=== expected\n%s=== %s\n%s", target, want, target, got)
		}
		if first == nil {
			first = &got
		} else if got != *first {
			t.Errorf("%s differs from %s\n=== %s\n%s=== %s\n%s", target, targets[0], targets[0], *first, target, got)
		}
	}
}

// HasDirective reports whether src carries "// goalchemy:<name>".
func HasDirective(src, name string) bool {
	for _, m := range directive.FindAllStringSubmatch(src, -1) {
		if m[1] == name {
			return true
		}
	}
	return false
}
