package integration

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/eugenioenko/goalchemy/internal/driver"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGeneratedRustHostFatalCleanup(t *testing.T) {
	for _, mode := range []string{"recursion", "cooperative-recursion", "panic", "mutex-fatal", "source-step-fault", "deferred-source-step-fault", "main-return", "adapter-fault", "deadlock"} {
		t.Run(mode, func(t *testing.T) {
			body := map[string]string{
				"recursion":                  "defer func(){recover()}(); recur(1)",
				"cooperative-recursion":      "defer func(){recover()}(); coRecur(1)",
				"panic":                      "panic(\"real source panic\")",
				"mutex-fatal":                "var m sync.Mutex;m.Unlock()",
				"source-step-fault":          "time.Sleep(446)",
				"deferred-source-step-fault": "defer func(){time.Sleep(446)}();return",
				"main-return":                "return",
				"adapter-fault":              "time.Sleep(447)",
				"deadlock":                   "select{}",
			}[mode]
			source := `package main
import("github.com/eugenioenko/goalchemy/lib/time";"github.com/eugenioenko/goalchemy/lib/sync")
func recur(i int)int{return recur(i+1)+i}
func coRecur(i int)int{time.Sleep(0);return coRecur(i+1)+i}
func main(){var unused sync.Mutex;_=unused;go func(){time.Sleep(444)}();time.Sleep(445);` + body + `}`
			out := rustHostFixture(t, source)
			marker := filepath.Join(t.TempDir(), "cleanup-"+mode)
			rustHostInject(t, filepath.Join(out, "src/main.rs"), "mod rt;", "mod rt;\nmod test_only_fatal;")
			rustHostInject(t, filepath.Join(out, "src/main.rs"), "    run_main(", "    report_host_result(run_host());\n}\nfn unchanged_virtual_main() {\n    run_main(")
			rustHostInject(t, filepath.Join(out, "src/rt/runtime/std_time_sleep.rs"), "    check_task(t);", "    check_task(t);\n    if (444..=447).contains(&d.i()) { crate::test_only_fatal::sleep(t,d.i());return; }")
			adapter := strings.NewReplacer("CLEANUP_FILE", marker, "MODE", mode).Replace(rustTestOnlyFatal)
			if e := os.WriteFile(filepath.Join(out, "src/test_only_fatal.rs"), []byte(adapter), 0600); e != nil {
				t.Fatal(e)
			}
			rustHostCommand(t, out, "rustc", "--edition", "2021", "-Awarnings", "-C", "opt-level=2", "-o", "fatal", "src/main.rs")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, filepath.Join(out, "fatal"))
			cmd.Dir = out
			cmd.Env = append(driver.ToolEnv(), "GOALCHEMY_GC_THRESHOLD=1")
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
			if e = cmd.Start(); e != nil {
				t.Fatal(e)
			}
			barrier := make(chan string, 1)
			go func() {
				s := bufio.NewScanner(output)
				if s.Scan() {
					barrier <- s.Text()
				} else {
					barrier <- "missing actual ACK wait"
				}
			}()
			select {
			case line := <-barrier:
				wantBarrier := "owner waiting for cleanup ACK"
				if mode == "deadlock" {
					wantBarrier = "pending native work prevents deadlock"
				}
				if line != wantBarrier {
					t.Fatal(line)
				}
			case <-ctx.Done():
				t.Fatal("actual ACK wait timeout")
			}
			if _, e = os.Stat(marker); !os.IsNotExist(e) {
				t.Fatal("cleanup marker existed before parent release")
			}
			if _, e = fmt.Fprintln(input, "release native resources"); e != nil {
				t.Fatal(e)
			}
			input.Close()
			e = cmd.Wait()
			wantExit := 2
			wantStderr := ""
			switch mode {
			case "recursion", "cooperative-recursion":
				wantStderr = "runtime: goroutine stack exceeds limit\nfatal error: stack overflow\n"
			case "panic":
				wantStderr = "panic: real source panic\n"
			case "mutex-fatal":
				wantStderr = "fatal error: sync: unlock of unlocked mutex\n"
			case "source-step-fault", "deferred-source-step-fault":
				wantExit = 101
				wantStderr = "goalchemy runtime fault: unexpected source step\n"
			case "adapter-fault":
				wantExit = 101
				wantStderr = "goalchemy runtime fault: applicable adapter fault\n"
			case "deadlock":
				wantStderr = "fatal error: all goroutines are asleep - deadlock!\n"
			case "main-return":
				wantExit = 0
			}
			if wantExit == 0 {
				if e != nil {
					t.Fatalf("main return %v %s", e, stderr.String())
				}
			} else {
				exit, ok := e.(*exec.ExitError)
				if !ok || exit.ExitCode() != wantExit {
					t.Fatalf("fatal exit %v stderr %q", e, stderr.String())
				}
			}
			if stderr.String() != wantStderr {
				t.Fatalf("cleanup-before-report stderr %q want %q", stderr.String(), wantStderr)
			}
			b, e := os.ReadFile(marker)
			if e != nil || string(b) != "mode="+mode+"; input/key released; "+map[bool]string{true: "pending wait", false: "owner ACK wait"}[mode == "deadlock"]+" observed\n" {
				t.Fatalf("fresh exact cleanup marker %q %v", b, e)
			}
		})
	}
}

const rustTestOnlyFatal = `use crate::rt::*;
use std::sync::{Arc,Mutex as NativeMutex,Condvar,atomic::Ordering};
use std::io::Write;
static STATE:(NativeMutex<(bool,bool)>,Condvar)=(NativeMutex::new((false,false)),Condvar::new());
fn cancel_result(_:V)->Vec<V>{vec![]}
pub fn sleep(t:&Rc<Task>,mode:i64){
 if mode==446{host_fault("unexpected source step")}
 if mode==447{let tok=register_host(t,host_boundary(background(),None),vec![],cancel_result,|_|vec![],||{},||{});tok.fault("applicable adapter fault");tok.acknowledge_cleanup();return}
 let token=register_host(t,host_boundary(background(),None),vec![byte_array(b"traced panic root".to_vec())],cancel_result,|_|vec![],move||{if mode==444{let mut s=STATE.0.lock().unwrap();s.1=true;STATE.1.notify_all();}},||{collect();});
 let inspect=token.clone();
 if mode==445{launch_host(token,||{let mut s=STATE.0.lock().unwrap();while !s.0{s=STATE.1.wait(s).unwrap()}vec![]});return}
 launch_host(token,move||{
  let input=vec![255u8;65536];let key_snapshot=vec![128u8;1024];
  {let mut s=STATE.0.lock().unwrap();s.0=true;STATE.1.notify_all();while !s.1 && "MODE"!="deadlock"{s=STATE.1.wait(s).unwrap()}}
  let deadlock="MODE"=="deadlock";
  while !(if deadlock{inspect.mailbox.waiting_driver.load(Ordering::Acquire)}else{inspect.mailbox.waiting_cleanup.load(Ordering::Acquire)}){std::thread::yield_now();}
  if deadlock{println!("pending native work prevents deadlock")}else{println!("owner waiting for cleanup ACK")};std::io::stdout().flush().unwrap();let mut line=String::new();std::io::stdin().read_line(&mut line).unwrap();assert_eq!(line,"release native resources\n");assert_eq!(input[65535],255);assert_eq!(key_snapshot[1023],128);
  drop(input);drop(key_snapshot);std::fs::write("CLEANUP_FILE",format!("mode=MODE; input/key released; {} observed\n",if deadlock{"pending wait"}else{"owner ACK wait"})).unwrap();vec![]
 });
}
`
