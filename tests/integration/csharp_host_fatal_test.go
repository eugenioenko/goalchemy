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

// Recursion is compiler output and reaches Program.enterSource; this never
// intentionally exhausts the uncatchable native CLR stack.
func TestGeneratedCSharpHostFatalCleanup(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"overflow", "panic", "mutex-fatal"} {
		t.Run(mode, func(t *testing.T) {
			fixture, out := t.TempDir(), t.TempDir()
			marker := filepath.Join(t.TempDir(), "cleanup-"+mode)
			ending := map[string]string{"overflow": "println(recurse(1))", "panic": "panic(\"source failure\")", "mutex-fatal": "var m sync.Mutex;m.Unlock()"}[mode]
			imports := "\"github.com/eugenioenko/goalchemy/lib/time\""
			if mode == "mutex-fatal" {
				imports += ";\"github.com/eugenioenko/goalchemy/lib/sync\""
			}
			source := fmt.Sprintf("package main\nimport(%s)\nfunc recurse(n int)int{return recurse(n+1)+n}\nfunc main(){go func(){time.Sleep(777)}();time.Sleep(778);%s}\n", imports, ending)
			for name, data := range map[string]string{"go.mod": fmt.Sprintf("module csharpfatal\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root), "main.go": source} {
				if err := os.WriteFile(filepath.Join(fixture, name), []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if ds := testutil.CompileGate(fixture, "csharp", out, "cooperative"); len(ds) > 0 {
				t.Fatal(ds)
			}
			emitted, err := testutil.GeneratedSource(out, ".cs")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(emitted), "using var _sourceDepth = Program.enterSource();") {
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
			inject(filepath.Join(out, "rt/runtime/StdTimeSleep.cs"), "        t.rv = Array.Empty<object>();", `        if(d==777){TestOnlyFatal.pending(t);return;}if(d==778){yieldTask(t);return;}
        t.rv = Array.Empty<object>();`)
			adapter := strings.NewReplacer("MODE_LITERAL", fmt.Sprintf("%q", mode), "MARKER_LITERAL", fmt.Sprintf("%q", marker)).Replace(csharpTestOnlyFatal)
			if err := os.WriteFile(filepath.Join(out, "rt/runtime/TestOnlyFatal.cs"), []byte(adapter), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(out, "FatalConsumer.cs"), []byte("using Rt;public static class FatalConsumer{public static void Main(string[] args){GoProgram.runHost().GetAwaiter().GetResult();}}"), 0600); err != nil {
				t.Fatal(err)
			}
			inject(filepath.Join(out, "run.sh"), "-optimize+", "-main:FatalConsumer -optimize+")
			inject(filepath.Join(out, "run.sh"), "-out:bin/main.dll $refs", "-out:bin/main.dll $refs FatalConsumer.cs rt/runtime/TestOnlyFatal.cs")
			ctx, stop := context.WithTimeout(context.Background(), 60*time.Second)
			defer stop()
			cmd := exec.CommandContext(ctx, "sh", "run.sh")
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
			want := map[string]string{"overflow": "runtime: goroutine stack exceeds limit\nfatal error: stack overflow\n", "panic": "panic: source failure\n", "mutex-fatal": "fatal error: sync: unlock of unlocked mutex\n"}[mode]
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

const csharpTestOnlyFatal = `namespace Rt;
/// TEST ONLY cleanup gate injected into copied compiler output.
public static class TestOnlyFatal{
 public static void pending(GoTask t){
  var owner=R.sched;var requested=new System.Threading.ManualResetEventSlim();
  byte[] input={0,255,128};var lease=(byte[])R.snapshot(input);input[1]=3;
  var token=owner.registerHost(t,requested.Set);
  _=System.Threading.Tasks.Task.Run(()=>{
   if(!requested.Wait(10000))throw new Exception("fatal cancellation gate timed out");
   if(!System.Threading.SpinWait.SpinUntil(()=>{lock(owner.mail)return owner.closed&&owner.mail.waiting&&owner.mail.retiring;},10000))throw new Exception("owner not in ACK wait");
   if(lease[1]!=255)throw new Exception("input retention");Console.WriteLine("owner waiting for cleanup ACK");Console.Out.Flush();
   if(Console.ReadLine()!="release")throw new Exception("parent cleanup gate");lease=null;requested.Dispose();
   File.WriteAllText(MARKER_LITERAL,MODE_LITERAL+":clean");token.acknowledgeCleanup();
  });
 }
}`

func TestGeneratedCSharpSequentialFatal(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	fixture, out := t.TempDir(), t.TempDir()
	for name, data := range map[string]string{"go.mod": fmt.Sprintf("module csharpseqfatal\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root), "main.go": "package main\nimport \"github.com/eugenioenko/goalchemy/lib/sync\"\nfunc main(){var m sync.Mutex;m.Unlock()}\n"} {
		if err := os.WriteFile(filepath.Join(fixture, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if ds := testutil.CompileGate(fixture, "csharp", out, "sequential"); len(ds) > 0 {
		t.Fatal(ds)
	}
	cmd := exec.Command("sh", "run.sh")
	cmd.Dir = out
	cmd.Env = driver.ToolEnv()
	result, err := cmd.CombinedOutput()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 2 || string(result) != "fatal error: sync: unlock of unlocked mutex\n" {
		t.Fatalf("sequential fatal behavior: %v %q", err, result)
	}
}
