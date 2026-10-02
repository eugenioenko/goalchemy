package contracts

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eugenioenko/goalchemy/internal/driver"
)

func TestJavaHostOperations(t *testing.T) {
	out := t.TempDir()
	var sources []string
	for _, dir := range []string{"targets/java/types", "targets/java/runtime"} {
		if err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".java") {
				sources = append(sources, path)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	sources = append(sources, filepath.Join(root, "targets/java/tests/HostOperationsTest.java"))
	env := driver.ToolEnv()
	bin := ""
	for _, item := range env {
		if strings.HasPrefix(item, "JAVA_HOME=") {
			bin = filepath.Join(strings.TrimPrefix(item, "JAVA_HOME="), "bin")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, filepath.Join(bin, "javac"), append([]string{"-d", out}, sources...)...)
	cmd.Env = env
	if result, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("JDK21 compile: %v\n%s", err, result)
	}
	cmd = exec.CommandContext(ctx, filepath.Join(bin, "java"), "-ea", "-cp", out, "rt.HostOperationsTest")
	cmd.Env = env
	if result, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("actual JVM host lifecycle: %v\n%s", err, result)
	} else {
		t.Log(string(result))
	}
	for _, mode := range []string{"panic", "deadlock", "overflow", "mutex-fatal"} {
		t.Run(mode, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "cleanup-"+mode)
			cmd := exec.CommandContext(ctx, filepath.Join(bin, "java"), "-ea", "-cp", out, "rt.HostOperationsTest", mode, marker)
			cmd.Env = env
			var result []byte
			var err error
			if mode == "deadlock" {
				result, err = cmd.CombinedOutput()
			} else {
				cmd.Args = append(cmd.Args, "gate")
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
						barrier <- "missing cancellation barrier"
					}
				}()
				select {
				case line := <-barrier:
					if line != "native cancellation requested" {
						t.Fatal(line)
					}
				case <-time.After(10 * time.Second):
					t.Fatal("fatal shutdown did not request native cancellation")
				}
				terminal := make(chan error, 1)
				go func() { terminal <- cmd.Wait() }()
				select {
				case e := <-terminal:
					t.Fatalf("process returned before cleanup ACK: %v", e)
				default:
				}
				if _, e := os.Stat(marker); !os.IsNotExist(e) {
					t.Fatalf("cleanup marker exists before release: %v", e)
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
				t.Fatalf("actual fatal %s must exit2: %v\n%s", mode, err, result)
			}
			want := map[string]string{"panic": "panic: source failure\n", "deadlock": "fatal error: all goroutines are asleep - deadlock!\n", "overflow": "runtime: goroutine stack exceeds limit\nfatal error: stack overflow\n", "mutex-fatal": "fatal error: sync: unlock of unlocked mutex\n"}[mode]
			if string(result) != want {
				t.Fatalf("fatal report: %s", result)
			}
			data, err := os.ReadFile(marker)
			if err != nil || string(data) != mode+":clean" {
				t.Fatalf("fresh cleanup marker before exit: %q %v", data, err)
			}
		})
	}
}
