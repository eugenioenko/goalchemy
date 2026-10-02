package contracts

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/eugenioenko/goalchemy/internal/driver"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestPythonHostOperations(t *testing.T) {
	script := filepath.Join(root, "targets/python/tests/host_operations_test.py")
	env := driver.ToolEnv()
	ctx, stop := context.WithTimeout(context.Background(), 120*time.Second)
	defer stop()
	cmd := exec.CommandContext(ctx, "python3", script)
	cmd.Env = env
	if result, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("actual CPython host lifecycle: %v\n%s", err, result)
	} else {
		t.Log(string(result))
	}
	for _, mode := range []string{"panic", "deadlock", "mutex-fatal"} {
		t.Run(mode, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "cleanup-"+mode)
			cmd := exec.CommandContext(ctx, "python3", script, mode, marker)
			cmd.Env = env
			var result []byte
			var err error
			if mode == "deadlock" {
				result, err = cmd.CombinedOutput()
			} else {
				input, e := cmd.StdinPipe()
				if e != nil {
					t.Fatal(e)
				}
				output, e := cmd.StdoutPipe()
				if e != nil {
					t.Fatal(e)
				}
				var stderr bytes.Buffer
				cmd.Stderr = &stderr
				if e := cmd.Start(); e != nil {
					t.Fatal(e)
				}
				barrier := make(chan string, 1)
				go func() {
					scanner := bufio.NewScanner(output)
					if scanner.Scan() {
						barrier <- scanner.Text()
					} else {
						barrier <- "missing cleanup barrier"
					}
				}()
				select {
				case line := <-barrier:
					if line != "owner waiting for cleanup ACK" {
						t.Fatal(line)
					}
				case <-time.After(10 * time.Second):
					t.Fatal("driver did not reach cleanup wait")
				}
				terminal := make(chan error, 1)
				go func() { terminal <- cmd.Wait() }()
				select {
				case e := <-terminal:
					t.Fatalf("exit before cleanup: %v", e)
				default:
				}
				if _, e := os.Stat(marker); !os.IsNotExist(e) {
					t.Fatal("marker before cleanup", e)
				}
				if _, e := fmt.Fprintln(input, "release"); e != nil {
					t.Fatal(e)
				}
				input.Close()
				err = <-terminal
				result = stderr.Bytes()
			}
			exit, ok := err.(*exec.ExitError)
			if !ok || exit.ExitCode() != 2 {
				t.Fatalf("fatal must exit2: %v\n%s", err, result)
			}
			want := map[string]string{"panic": "panic: source failure\n", "deadlock": "fatal error: all goroutines are asleep - deadlock!\n", "mutex-fatal": "fatal error: sync: unlock of unlocked mutex\n"}[mode]
			if string(result) != want {
				t.Fatalf("exact report: %q", result)
			}
			data, err := os.ReadFile(marker)
			if err != nil || string(data) != mode+":clean" {
				t.Fatalf("fresh exact cleanup: %q %v", data, err)
			}
		})
	}
}
