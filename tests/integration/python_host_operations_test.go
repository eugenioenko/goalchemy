package integration

import (
	"context"
	"encoding/base64"
	"fmt"
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

	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/testutil"
)

// These visibly test-only native adapters are copied into compiler output.
// They are not production lib.http.do and add no capability mapping.
func TestGeneratedPythonHostOperations(t *testing.T) {
	var normal, release, cancel, tlsContact atomic.Int32
	bodyRelease := make(chan struct{})
	var normalMu sync.Mutex
	var normalGate chan struct{}
	canceled := make(chan struct{})
	var releaseOnce, cancelOnce sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/normal":
			normal.Add(1)
			normalMu.Lock()
			normalGate = make(chan struct{})
			gate := normalGate
			normalMu.Unlock()
			w.Write([]byte{0, 255, 128, 3})
			w.(http.Flusher).Flush()
			select {
			case <-gate:
			case <-bodyRelease:
			case <-r.Context().Done():
			}
		case "/release":
			release.Add(1)
			normalMu.Lock()
			close(normalGate)
			normalMu.Unlock()
			w.Write([]byte("released"))
		case "/cancel":
			cancel.Add(1)
			w.Write([]byte{0, 255, 128, 3})
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			cancelOnce.Do(func() { close(canceled) })
		default:
			t.Errorf("unexpected real server path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	defer releaseOnce.Do(func() { close(bodyRelease) })
	tls := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { tlsContact.Add(1); w.Write([]byte{0, 255, 128, 3}) }))
	defer tls.Close()
	fixture, out := t.TempDir(), t.TempDir()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source := `package main
import (
 "github.com/eugenioenko/goalchemy/lib/context"
 "github.com/eugenioenko/goalchemy/lib/time"
)
var counter int
func init(){ counter++;println("init",counter) }
func main(){
 counter++;println("main",counter)
 buffered:=make(chan *int,1)
 buffered<-nil
 println(<-buffered==nil)
 var missing context.Context
 println(missing==nil,nil==missing)
 go func(){time.Sleep(124);println("source progress");time.Sleep(125)}()
 time.Sleep(123)
 println("transport returned")
 ctx,stop:=context.WithCancel(context.Background())
 go func(){time.Sleep(124);println("cancel progress");stop()}()
 time.Sleep(126)
 println(ctx.Err()==context.Canceled)
 time.Sleep(128)
 time.Sleep(129)
 expired,ce:=context.WithTimeout(context.Background(),-1)
 defer ce()
 child,cc:=context.WithTimeout(expired,time.Second)
 defer cc()
 println(expired.Err()==context.DeadlineExceeded,child.Err()==context.DeadlineExceeded)
}`
	for name, data := range map[string]string{"go.mod": fmt.Sprintf("module pythonhostprobe\n\ngo 1.25\n\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root), "main.go": source} {
		if err := os.WriteFile(filepath.Join(fixture, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if ds := testutil.CompileGate(fixture, "python", out, "cooperative"); len(ds) > 0 {
		t.Fatal(ds)
	}

	env := driver.ToolEnv()
	ctx, stop := context.WithTimeout(context.Background(), 120*time.Second)
	defer stop()
	run := func(file string) string {
		t.Helper()
		cmd := exec.CommandContext(ctx, "python3", file)
		cmd.Dir = out
		cmd.Env = env
		result, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("actual emitted CPython: %v\n%s", err, result)
		}
		return string(result)
	}
	baseline := run("main.py")
	if baseline != "init 1\nmain 2\ntrue\ntrue true\ntransport returned\nsource progress\ncancel progress\ntrue\ntrue true\n" {
		t.Fatalf("unchanged emitted virtual behavior: %q", baseline)
	}
	inject := func(path, marker, replacement string) {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(data), marker) != 1 {
			t.Fatalf("test-only replacement marker must be unique: %s %q", path, marker)
		}
		if err := os.WriteFile(path, []byte(strings.Replace(string(data), marker, replacement, 1)), 0600); err != nil {
			t.Fatal(err)
		}
	}
	inject(filepath.Join(out, "rt/runtime/std_time_sleep.py"), "    t.rv = []", `    if 123 <= d <= 129:
        from test_only_transport import sleep
        sleep(t, d)
        return
    t.rv = []`)
	inject(filepath.Join(out, "rt/runtime/std_context_with_cancel.py"), "    c = new_child(parent)", `    c = new_child(parent)
    import test_only_transport
    test_only_transport.context = c`)
	adapter := strings.NewReplacer("HTTP_URL", srv.URL, "TLS_URL", tls.URL, "TLS_CERT", base64.StdEncoding.EncodeToString(tls.Certificate().Raw)).Replace(pythonTestOnlyTransport)
	if err := os.WriteFile(filepath.Join(out, "test_only_transport.py"), []byte(adapter), 0600); err != nil {
		t.Fatal(err)
	}
	consumer := `import main, rt, test_only_transport as transport
import sys, threading
before=(sys.getrecursionlimit(), threading.stack_size())
main.runHost()
main.runHost()
assert before==(sys.getrecursionlimit(), threading.stack_size())
assert transport.cleaned==10 and transport.retained==0
assert rt.panic_state.get() is not None
print('PASS emitted Python host frames/repeated init/HTTP/TLS/cancellation/cleanup')
`
	if err := os.WriteFile(filepath.Join(out, "consumer.py"), []byte(consumer), 0600); err != nil {
		t.Fatal(err)
	}
	actual := run("consumer.py")
	want := "init 1\nmain 2\ntrue\ntrue true\nsource progress\ntransport returned\ncancel progress\ntrue\ntrue true\ninit 3\nmain 4\ntrue\ntrue true\nsource progress\ntransport returned\ncancel progress\ntrue\ntrue true\nPASS emitted Python host frames/repeated init/HTTP/TLS/cancellation/cleanup\n"
	if actual != want {
		t.Fatalf("actual emitted host output: %q", actual)
	}
	select {
	case <-canceled:
	case <-time.After(10 * time.Second):
		t.Fatal("canceled transport did not release server")
	}
	if normal.Load() != 2 || release.Load() != 2 || cancel.Load() != 2 || tlsContact.Load() != 2 {
		t.Fatalf("exact genuine server contacts normal=%d release=%d cancel=%d tls=%d", normal.Load(), release.Load(), cancel.Load(), tlsContact.Load())
	}
	t.Log(actual)
}

// Visibly TEST ONLY: deliberately omits the production lib.http.do contract.
const pythonTestOnlyTransport = `import http.client, ssl, socket, base64, threading
from urllib.parse import urlsplit
import rt
context=None
cleaned=0
retained=0
started=None
lock=threading.Lock()

def sleep(t,d):
 global started,retained
 if d==124:
  gate=started
  token=rt.register_host(t)
  def wait_start():
   try:
    if not gate.wait(10):raise RuntimeError('native/server start barrier')
    token.publish([])
   except BaseException as e:token.publish(fault=str(e))
   finally:token.acknowledge_cleanup()
  threading.Thread(target=wait_start).start()
  return
 if d in (123,126):started=threading.Event()
 gate=started
 canceled=threading.Event()
 state={'socket':None}
 def cancel():
  canceled.set()
  with lock:
   sock=state['socket']
   if sock is not None:
    try:sock.shutdown(socket.SHUT_RDWR)
    except OSError:pass
 source=context if d==126 else None
 token=rt.register_host(t,source,cancel=cancel,cancel_result=lambda err:[])
 original=bytearray([0,255,128,3]);inputs=rt.snapshot(original);original[1]=1
 with lock:retained+=1
 lease={'input':inputs}
 def work():
  global cleaned,retained
  inputs=lease.pop('input')
  conn=response=sock=None;failure=None
  try:
   assert inputs==bytes([0,255,128,3])
   url='HTTP_URL/normal' if d==123 else 'HTTP_URL/release' if d==125 else 'HTTP_URL/cancel' if d==126 else 'TLS_URL'
   parsed=urlsplit(url)
   if parsed.scheme=='https':
    trust=ssl.create_default_context()
    if d!=129:trust.load_verify_locations(cadata=ssl.DER_cert_to_PEM_cert(base64.b64decode('TLS_CERT')))
    conn=http.client.HTTPSConnection(parsed.hostname,parsed.port,context=trust,timeout=10)
   else:conn=http.client.HTTPConnection(parsed.hostname,parsed.port,timeout=10)
   conn.connect();sock=conn.sock
   with lock:state['socket']=sock
   if canceled.is_set():cancel()
   conn.request('GET',parsed.path or '/')
   response=conn.getresponse()
   if d==129:raise RuntimeError('untrusted TLS accepted')
   first=response.read(8 if d==125 else 4)
   if d!=125:assert first==inputs
   if d in (123,126):gate.set()
   response.read()
  except ssl.SSLError as e:
   if d!=129:failure='unexpected TLS rejection: '+str(e)
  except BaseException as e:
   if not (d==126 and canceled.is_set()):failure='test-only transport: '+str(e)
  finally:
   if response is not None:response.close()
   if conn is not None:conn.close()
   if sock is not None:sock.close()
   with lock:
    state['socket']=None;retained-=1;cleaned+=1
   inputs=None;response=conn=sock=None
   token.publish([],fault=failure)
   token.acknowledge_cleanup()
 threading.Thread(target=work).start()
`
