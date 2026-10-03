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

func TestCSharpHostOperations(t *testing.T) {
	out := t.TempDir()
	env := driver.ToolEnv()
	droot := ""
	for _, entry := range env {
		if strings.HasPrefix(entry, "DOTNET_ROOT=") {
			droot = strings.TrimPrefix(entry, "DOTNET_ROOT=")
		}
	}
	if droot == "" {
		t.Fatal("pinned .NET8 unavailable")
	}
	sdks, err := filepath.Glob(filepath.Join(droot, "sdk", "8.*", "Roslyn", "bincore", "csc.dll"))
	if err != nil || len(sdks) == 0 {
		t.Fatal(".NET8 compiler missing", err)
	}
	refs, err := filepath.Glob(filepath.Join(droot, "packs", "Microsoft.NETCore.App.Ref", "8.*", "ref", "net8.0", "*.dll"))
	if err != nil || len(refs) == 0 {
		t.Fatal(".NET8 refs missing", err)
	}
	args := []string{sdks[len(sdks)-1], "-nologo", "-noconfig", "-nostdlib", "-langversion:12", "-nullable:disable", "-out:" + filepath.Join(out, "host.dll"), "-main:Rt.HostOperationsTest"}
	for _, ref := range refs {
		args = append(args, "-r:"+ref)
	}
	for _, dir := range []string{"targets/csharp/types", "targets/csharp/runtime"} {
		if err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && strings.HasSuffix(path, ".cs") {
				args = append(args, path)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	args = append(args, filepath.Join(root, "targets/csharp/tests/HostOperationsTest.cs"))
	ctx, stop := context.WithTimeout(context.Background(), 120*time.Second)
	defer stop()
	hashing := csharpChecksumDLL(t, ctx)
	args = append(args, "-r:"+hashing)
	dll, err := os.ReadFile(hashing)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "System.IO.Hashing.dll"), dll, 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(ctx, filepath.Join(droot, "dotnet"), args...)
	cmd.Env = env
	if result, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("actual .NET8 compiler: %v\n%s", err, result)
	}
	if err := os.WriteFile(filepath.Join(out, "host.runtimeconfig.json"), []byte(`{"runtimeOptions":{"tfm":"net8.0","framework":{"name":"Microsoft.NETCore.App","version":"8.0.0"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	cmd = exec.CommandContext(ctx, filepath.Join(droot, "dotnet"), filepath.Join(out, "host.dll"))
	cmd.Env = env
	if result, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("actual CLR host lifecycle: %v\n%s", err, result)
	} else {
		t.Log(string(result))
	}
	for _, mode := range []string{"panic", "deadlock", "mutex-fatal"} {
		t.Run(mode, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "cleanup-"+mode)
			cmd := exec.CommandContext(ctx, filepath.Join(droot, "dotnet"), filepath.Join(out, "host.dll"), mode, marker)
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
