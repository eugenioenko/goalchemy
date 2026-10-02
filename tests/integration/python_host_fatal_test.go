package integration

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/testutil"
)

// Genuine emitted ordinary recursion exercises both the generated source-call
// guard and actual CPython RecursionError, each after a native lease start barrier.
func TestGeneratedPythonHostFatalCleanup(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"overflow", "recursion-error", "panic", "mutex-fatal"} {
		t.Run(mode, func(t *testing.T) {
			fixture, out := t.TempDir(), t.TempDir()
			marker := filepath.Join(t.TempDir(), "cleanup-"+mode)
			ending := map[string]string{"overflow": "println(recurse(1))", "recursion-error": "println(recurse(1))", "panic": "panic(\"source failure\")", "mutex-fatal": "var m sync.Mutex;m.Unlock()"}[mode]
			imports := "\"github.com/eugenioenko/goalchemy/lib/time\""
			if mode == "mutex-fatal" {
				imports += ";\"github.com/eugenioenko/goalchemy/lib/sync\""
			}
			source := fmt.Sprintf("package main\nimport(%s)\nfunc recurse(n int)int{return recurse(n+1)+n}\nfunc main(){go func(){time.Sleep(777)}();time.Sleep(778);%s}\n", imports, ending)
			for name, data := range map[string]string{"go.mod": fmt.Sprintf("module pythonfatal\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root), "main.go": source} {
				if err := os.WriteFile(filepath.Join(fixture, name), []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if ds := testutil.CompileGate(fixture, "python", out, "cooperative"); len(ds) > 0 {
				t.Fatal(ds)
			}
			emitted, err := os.ReadFile(filepath.Join(out, "main.py"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(emitted), "@rt.source_guard") {
				t.Fatal("generated ordinary recursion missing guard")
			}
			inject := func(path, marker, replacement string) {
				t.Helper()
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Count(string(data), marker) != 1 {
					t.Fatal("unique test-only marker missing", path)
				}
				if err := os.WriteFile(path, []byte(strings.Replace(string(data), marker, replacement, 1)), 0600); err != nil {
					t.Fatal(err)
				}
			}
			inject(filepath.Join(out, "rt/runtime/std_time_sleep.py"), "    t.rv = []", `    if d == 777:
        from test_only_fatal import pending
        pending(t)
        return
    if d == 778:
        yield_task(t)
        return
    t.rv = []`)
			adapter := strings.NewReplacer("MODE_LITERAL", fmt.Sprintf("%q", mode), "MARKER_LITERAL", fmt.Sprintf("%q", marker)).Replace(pythonTestOnlyFatal)
			if err := os.WriteFile(filepath.Join(out, "test_only_fatal.py"), []byte(adapter), 0600); err != nil {
				t.Fatal(err)
			}
			consumer := "import main, sys\n"
			if mode == "recursion-error" {
				consumer += "sys.setrecursionlimit(100) # TEST ONLY: actual CPython RecursionError before source guard bound\n"
			}
			consumer += "main.runHost()\n"
			if err := os.WriteFile(filepath.Join(out, "consumer.py"), []byte(consumer), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, stop := context.WithTimeout(context.Background(), 60*time.Second)
			defer stop()
			cmd := exec.CommandContext(ctx, "python3", "consumer.py")
			cmd.Dir = out
			cmd.Env = driver.ToolEnv()
			input, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			output, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			barrier := make(chan string, 1)
			go func() {
				scanner := bufio.NewScanner(output)
				if scanner.Scan() {
					barrier <- scanner.Text()
				} else {
					barrier <- "missing owner cleanup wait"
				}
			}()
			select {
			case line := <-barrier:
				if line != "owner waiting for cleanup ACK" {
					t.Fatalf("barrier %q\n%s", line, stderr.Bytes())
				}
			case <-time.After(20 * time.Second):
				t.Fatal("generated fatal never reached cleanup ACK wait")
			}
			terminal := make(chan error, 1)
			go func() { terminal <- cmd.Wait() }()
			select {
			case e := <-terminal:
				t.Fatalf("exit before cleanup: %v", e)
			default:
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatal("marker before gate", err)
			}
			fmt.Fprintln(input, "release")
			input.Close()
			err = <-terminal
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 2 {
				t.Fatalf("fatal exit2: %v\n%s", err, stderr.Bytes())
			}
			want := map[string]string{"overflow": "runtime: goroutine stack exceeds limit\nfatal error: stack overflow\n", "recursion-error": "runtime: goroutine stack exceeds limit\nfatal error: stack overflow\n", "panic": "panic: source failure\n", "mutex-fatal": "fatal error: sync: unlock of unlocked mutex\n"}[mode]
			if stderr.String() != want {
				t.Fatalf("exact generated fatal report: %q", stderr.String())
			}
			data, err := os.ReadFile(marker)
			if err != nil || string(data) != mode+":clean" {
				t.Fatalf("fresh cleanup before fatal exit: %q %v", data, err)
			}
		})
	}
}

const pythonTestOnlyFatal = `import threading,sys
from pathlib import Path
import rt
# TEST ONLY: resource retention/cleanup gate, confined to copied compiler output.
def pending(t):
 owner=rt.sched();requested=threading.Event();waiting=threading.Event()
 owner.retiring_wait=waiting.set
 original=bytearray([0,255,128]);lease={'input':rt.snapshot(original)};original[1]=3
 token=rt.register_host(t,cancel=requested.set)
 started=threading.Event()
 def worker():
  started.set()
  assert requested.wait(10) and waiting.wait(10)
  assert lease['input'][1]==255
  print('owner waiting for cleanup ACK',flush=True)
  assert sys.stdin.readline().strip()=='release'
  lease.clear()
  Path(MARKER_LITERAL).write_text(MODE_LITERAL+':clean')
  token.acknowledge_cleanup()
 threading.Thread(target=worker).start()
 assert started.wait(10)
`

func TestGeneratedPythonSequentialFatal(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	fixture, out := t.TempDir(), t.TempDir()
	for name, data := range map[string]string{"go.mod": fmt.Sprintf("module pythonseqfatal\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root), "main.go": "package main\nimport \"github.com/eugenioenko/goalchemy/lib/sync\"\nfunc main(){var m sync.Mutex;m.Unlock()}\n"} {
		if err := os.WriteFile(filepath.Join(fixture, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if ds := testutil.CompileGate(fixture, "python", out, "sequential"); len(ds) > 0 {
		t.Fatal(ds)
	}
	cmd := exec.Command("python3", "main.py")
	cmd.Dir = out
	cmd.Env = driver.ToolEnv()
	result, err := cmd.CombinedOutput()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 2 || string(result) != "fatal error: sync: unlock of unlocked mutex\n" {
		t.Fatalf("sequential fatal behavior: %v %q", err, result)
	}
}
