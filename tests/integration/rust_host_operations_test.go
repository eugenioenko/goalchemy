package integration

import (
	"context"
	"encoding/pem"
	"fmt"
	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/testutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func rustHostInject(t *testing.T, path, marker, replacement string) {
	t.Helper()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Count(string(b), marker) != 1 {
		t.Fatalf("nonunique test-only marker %s %q", path, marker)
	}
	if e = os.WriteFile(path, []byte(strings.Replace(string(b), marker, replacement, 1)), 0600); e != nil {
		t.Fatal(e)
	}
}
func rustHostFixture(t *testing.T, source string) string {
	t.Helper()
	fixture, out := t.TempDir(), t.TempDir()
	root, e := filepath.Abs("../..")
	if e != nil {
		t.Fatal(e)
	}
	for name, data := range map[string]string{"go.mod": fmt.Sprintf("module rusthostprobe\n\ngo 1.25\n\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root), "main.go": source} {
		if e = os.WriteFile(filepath.Join(fixture, name), []byte(data), 0600); e != nil {
			t.Fatal(e)
		}
	}
	if ds := testutil.CompileGate(fixture, "rust", out, "cooperative"); len(ds) > 0 {
		t.Fatal(ds)
	}
	return out
}
func rustHostCommand(t *testing.T, out string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = out
	cmd.Env = append(driver.ToolEnv(), "GOALCHEMY_GC_THRESHOLD=1")
	b, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("actual Rust %v: %v\n%s", args, e, b)
	}
	return string(b)
}

// Independent stdlib servers and a visibly test-only curl process adapter. This
// does not implement production HTTP validation/headers/limits/redirect policy.
func TestGeneratedRustHostOperations(t *testing.T) {
	var normal, release, cancel, contact, startedContact, tlsContact, nativeFaultContact atomic.Int32
	var mu sync.Mutex
	var body, seen, normalStarted chan struct{}
	var gates []chan struct{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/native-fault":
			nativeFaultContact.Add(1)
			w.Write([]byte{0, 255, 128, 3})
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		case "/reset":
			mu.Lock()
			body = make(chan struct{})
			seen = make(chan struct{})
			normalStarted = make(chan struct{})
			gates = append(gates, body, seen, normalStarted)
			mu.Unlock()
			w.Write([]byte("reset"))
		case "/normal":
			normal.Add(1)
			mu.Lock()
			gate := body
			started := normalStarted
			mu.Unlock()
			close(started)
			w.Write([]byte{0, 255, 128, 3})
			w.(http.Flusher).Flush()
			select {
			case <-gate:
			case <-r.Context().Done():
			}
		case "/started":
			startedContact.Add(1)
			mu.Lock()
			gate := normalStarted
			mu.Unlock()
			select {
			case <-gate:
			case <-r.Context().Done():
				return
			}
			w.Write([]byte("started"))
		case "/release":
			release.Add(1)
			mu.Lock()
			gate := body
			started := normalStarted
			mu.Unlock()
			select {
			case <-started:
			case <-r.Context().Done():
				return
			}
			close(gate)
			w.Write([]byte("released"))
		case "/cancel":
			cancel.Add(1)
			mu.Lock()
			gate := seen
			mu.Unlock()
			close(gate)
			w.Write([]byte{0, 255, 128, 3})
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		case "/contact":
			contact.Add(1)
			mu.Lock()
			gate := seen
			mu.Unlock()
			select {
			case <-gate:
			case <-r.Context().Done():
				return
			}
			w.Write([]byte("contact"))
		default:
			t.Errorf("unexpected HTTP path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer func() {
		mu.Lock()
		for _, c := range gates {
			select {
			case <-c:
			default:
				close(c)
			}
		}
		mu.Unlock()
		srv.Close()
	}()
	tls := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { tlsContact.Add(1); w.Write([]byte{0, 255, 128, 3}) }))
	defer tls.Close()
	out := rustHostFixture(t, `package main
import("github.com/eugenioenko/goalchemy/lib/context";"github.com/eugenioenko/goalchemy/lib/time")
var counter int
func init(){counter++;println("init",counter)}
func main(){
 counter++;println("main",counter)
 b:=make(chan *int,1);b<-nil;println(<-b==nil)
 var missing context.Context;println(missing==nil,nil==missing)
 go func(){time.Sleep(125);println("source progress");time.Sleep(124)}()
 time.Sleep(123);println("transport returned")
 ctx,stop:=context.WithCancel(context.Background())
 go func(){time.Sleep(127);stop()}()
 time.Sleep(126);println(ctx.Err()==context.Canceled)
 time.Sleep(128);time.Sleep(129);println("TLS done")
 expired,ce:=context.WithTimeout(context.Background(),-1);defer ce()
 child,cc:=context.WithTimeout(expired,time.Second);defer cc()
 println(expired.Err()==context.DeadlineExceeded,child.Err()==context.DeadlineExceeded)
}`)
	baseline := rustHostCommand(t, out, "sh", "run.sh")
	want := "init 1\nmain 2\ntrue\ntrue true\ntransport returned\nsource progress\nfalse\nTLS done\ntrue true\n"
	if baseline != want {
		t.Fatalf("unchanged emitted virtual output %q", baseline)
	}
	cert := filepath.Join(out, "test-only-ca.pem")
	if e := os.WriteFile(cert, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: tls.Certificate().Raw}), 0600); e != nil {
		t.Fatal(e)
	}
	rustHostInject(t, filepath.Join(out, "src/main.rs"), "mod rt;", "mod rt;\nmod test_only_transport;")
	rustHostInject(t, filepath.Join(out, "src/rt/runtime/std_time_sleep.rs"), "    check_task(t);", "    check_task(t);\n    if (123..=129).contains(&d.i()) { crate::test_only_transport::sleep(t, d.i()); return; }")
	rustHostInject(t, filepath.Join(out, "src/rt/runtime/std_context_with_cancel.rs"), "    let c = new_child(parent);", "    let c = new_child(parent);\n    crate::test_only_transport::remember_context(c.clone());")
	rustHostInject(t, filepath.Join(out, "src/main.rs"), "fn main() {", `fn main() {
    test_only_transport::begin_test_run(); run_host().unwrap();
    test_only_transport::begin_test_run(); run_host().unwrap();
    assert_eq!(test_only_transport::CLEANED.load(std::sync::atomic::Ordering::SeqCst), 14);
    assert_eq!(test_only_transport::RETAINED.load(std::sync::atomic::Ordering::SeqCst), 0);
    let fault=run_main_host(0,||{},test_only_transport::native_fault_entry);
    assert!(matches!(fault,Err(HostError::Fault(ref text)) if text=="forced post-acquisition native fault"));
    assert_eq!(test_only_transport::CLEANED.load(std::sync::atomic::Ordering::SeqCst),15);
    assert_eq!(test_only_transport::RETAINED.load(std::sync::atomic::Ordering::SeqCst),0);
}
fn unchanged_virtual_main() {`)
	adapter := strings.NewReplacer("HTTP_URL", srv.URL, "TLS_URL", tls.URL, "CA_FILE", cert).Replace(rustTestOnlyTransport)
	if e := os.WriteFile(filepath.Join(out, "src/test_only_transport.rs"), []byte(adapter), 0600); e != nil {
		t.Fatal(e)
	}
	output := rustHostCommand(t, out, "sh", "run.sh")
	expected := strings.Replace(strings.Replace(want, "transport returned\nsource progress", "source progress\ntransport returned", 1), "false\nTLS done", "true\nTLS done", 1)
	if !strings.HasPrefix(output, expected+expected) || !strings.Contains(output, "forced post-acquisition native fault") {
		t.Fatalf("emitted host output %q", output)
	}
	if normal.Load() != 2 || release.Load() != 2 || cancel.Load() != 2 || contact.Load() != 2 || startedContact.Load() != 2 || tlsContact.Load() != 2 || nativeFaultContact.Load() != 1 {
		t.Fatalf("exact independent contacts normal=%d release=%d cancel=%d contact=%d TLS=%d", normal.Load(), release.Load(), cancel.Load(), contact.Load(), tlsContact.Load())
	}
	// Real Cargo builds execute the same emitted host entry and native byte checks.
	rustHostCommand(t, out, "cargo", "build", "--release", "--offline")
}

const rustTestOnlyTransport = `use crate::rt::*;
use std::sync::{Arc, atomic::{AtomicBool, AtomicUsize, Ordering}};
use std::time::Duration;
use std::process::{Command, Stdio};
use std::io::Read;
pub static CLEANED: AtomicUsize=AtomicUsize::new(0);
pub static RETAINED: AtomicUsize=AtomicUsize::new(0);
struct NativeLease { child:std::process::Child, output:std::path::PathBuf, header:std::path::PathBuf, counted:bool }
impl Drop for NativeLease {fn drop(&mut self){let _=self.child.kill();let _=self.child.wait();let _=std::fs::remove_file(&self.output);let _=std::fs::remove_file(&self.header);if self.counted{RETAINED.fetch_sub(1,Ordering::SeqCst);CLEANED.fetch_add(1,Ordering::SeqCst);}}}
static SEQUENCE: AtomicUsize=AtomicUsize::new(0);
thread_local!{static CONTEXT:RefCell<V>=RefCell::new(V::Nil);}
pub fn begin_test_run(){let output=Command::new("curl").args(["--silent","--show-error","--max-time","10","HTTP_URL/reset"]).output().unwrap();assert!(output.status.success());assert_eq!(output.stdout,b"reset");}
fn fault_step(t:&Rc<Task>,_:&Rc<Frame>){sleep(t,130)}
pub fn native_fault_entry()->V{V::Frame(Frame::new(0,fault_step,None))}
pub fn remember_context(c:V){CONTEXT.with(|x|x.replace(c));}
fn cancel_result(_:V)->Vec<V>{Vec::new()}
pub fn sleep(t:&Rc<Task>,mode:i64){
 let context=if mode==126{CONTEXT.with(|c|c.borrow().clone())}else{background()};
 let canceled=Arc::new(AtomicBool::new(false));let cancel=canceled.clone();
 let boundary=host_boundary(context,Some(30_000_000_000));
 let token=register_host(t,boundary,vec![],cancel_result,move|wire|{
  match &wire[0]{HostWire::Bytes(b)=>match mode{123|128=>assert_eq!(b,&[0,255,128,3]),124=>assert_eq!(b,b"released"),127=>assert_eq!(b,b"contact"),125=>assert_eq!(b,b"started"),129=>panic!("untrusted TLS succeeded"),_=>{}},HostWire::Text(_)=>assert_eq!(mode,129),_=>panic!("wire")};Vec::new()
 },move||{cancel.store(true,Ordering::SeqCst);},||{});
 let path=match mode{123=>"/normal",124=>"/release",126=>"/cancel",127=>"/contact",125=>"/started",130=>"/native-fault",_=>"/"};
 let url=if mode>=128 && mode!=130{format!("TLS_URL{}",path)}else{format!("HTTP_URL{}",path)};
 launch_host(token,move||{
  let seq=SEQUENCE.fetch_add(1,Ordering::SeqCst);let output=std::env::temp_dir().join(format!("goalchemy-rust-test-{}-{}",std::process::id(),seq));let header=output.with_extension("headers");
  let mut command=Command::new("curl");command.args(["--silent","--show-error","--http1.1","--max-time","25","--output"]).arg(&output).arg("--dump-header").arg(&header);
  if mode==128{command.arg("--cacert").arg("CA_FILE");}
  command.arg(&url).stdout(Stdio::null()).stderr(Stdio::piped());
  let mut lease=NativeLease{child:command.spawn().unwrap(),output,header,counted:false};RETAINED.fetch_add(1,Ordering::SeqCst);lease.counted=true;
  if mode==130{while std::fs::read(&lease.header).map_or(true,|b|!b.windows(12).any(|x|x==b"HTTP/1.1 200")){std::thread::park_timeout(Duration::from_millis(1));}panic!("forced post-acquisition native fault");}
  let status=loop{if canceled.load(Ordering::SeqCst){let _=lease.child.kill();break lease.child.wait().unwrap()}if let Some(s)=lease.child.try_wait().unwrap(){break s}std::thread::park_timeout(Duration::from_millis(1));};
  let result=if status.success(){HostWire::Bytes(std::fs::read(&lease.output).unwrap())}else{let mut b=String::new();lease.child.stderr.take().unwrap().read_to_string(&mut b).unwrap();HostWire::Text(b)};
  drop(lease);vec![result]
 });
}
`
