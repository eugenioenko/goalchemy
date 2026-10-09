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
func TestGeneratedCSharpHostOperations(t *testing.T) {
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
	for name, data := range map[string]string{"go.mod": fmt.Sprintf("module csharphostprobe\n\ngo 1.25\n\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root), "main.go": source} {
		if err := os.WriteFile(filepath.Join(fixture, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if ds := testutil.CompileGate(fixture, "csharp", out, "cooperative"); len(ds) > 0 {
		t.Fatal(ds)
	}

	env := driver.ToolEnv()
	ctx, stop := context.WithTimeout(context.Background(), 120*time.Second)
	defer stop()
	run := func() string {
		t.Helper()
		cmd := exec.CommandContext(ctx, "sh", "run.sh")
		cmd.Dir = out
		cmd.Env = env
		result, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("actual CLR emitted output: %v\n%s", err, result)
		}
		return string(result)
	}
	baseline := run()
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
			t.Fatalf("test-only marker must be unique: %s %q", path, marker)
		}
		if err := os.WriteFile(path, []byte(strings.Replace(string(data), marker, replacement, 1)), 0600); err != nil {
			t.Fatal(err)
		}
	}
	inject(filepath.Join(out, "rt/runtime/StdTimeSleep.cs"), "        t.rv = Array.Empty<object>();", `        if(d>=123 && d<=129){TestOnlyTransport.sleep(t,d);return;}
        t.rv = Array.Empty<object>();`)
	inject(filepath.Join(out, "rt/runtime/StdContextWithCancel.cs"), "        var c = newChild(parent);", `        var c = newChild(parent);
        TestOnlyTransport.context=c;`)
	adapter := strings.NewReplacer("HTTP_URL", srv.URL, "TLS_URL", tls.URL, "TLS_CERT", base64.StdEncoding.EncodeToString(tls.Certificate().Raw)).Replace(csharpTestOnlyTransport)
	if err := os.WriteFile(filepath.Join(out, "rt/runtime/TestOnlyTransport.cs"), []byte(adapter), 0600); err != nil {
		t.Fatal(err)
	}
	consumer := `using System;using Rt;
public static class HostConsumer{
 public static void Main(string[] args){
  GoProgram.runHost().GetAwaiter().GetResult();GoProgram.runHost().GetAwaiter().GetResult();
  if(TestOnlyTransport.cleaned!=10||TestOnlyTransport.retained!=0)throw new Exception("exact native cleanup/input counts");
  if(Program.panicState()==null)throw new Exception("panic binding after reuse");
  Console.WriteLine("PASS emitted C# host frames/repeated init/HTTP/TLS/cancellation/cleanup");
 }
}`
	if err := os.WriteFile(filepath.Join(out, "HostConsumer.cs"), []byte(consumer), 0600); err != nil {
		t.Fatal(err)
	}
	inject(filepath.Join(out, "run.sh"), "-optimize+", "-main:HostConsumer -optimize+")
	inject(filepath.Join(out, "run.sh"), "-out:bin/main.dll $refs", "-out:bin/main.dll $refs HostConsumer.cs rt/runtime/TestOnlyTransport.cs")
	actual := run()
	want := "init 1\nmain 2\ntrue\ntrue true\nsource progress\ntransport returned\ncancel progress\ntrue\ntrue true\ninit 3\nmain 4\ntrue\ntrue true\nsource progress\ntransport returned\ncancel progress\ntrue\ntrue true\nPASS emitted C# host frames/repeated init/HTTP/TLS/cancellation/cleanup\n"
	if actual != want {
		t.Fatalf("actual emitted host output: %q", actual)
	}
	select {
	case <-canceled:
	case <-time.After(10 * time.Second):
		t.Fatal("native canceled request did not release independent server")
	}
	if normal.Load() != 2 || release.Load() != 2 || cancel.Load() != 2 || tlsContact.Load() != 2 {
		t.Fatalf("exact genuine server contacts normal=%d release=%d cancel=%d tls=%d", normal.Load(), release.Load(), cancel.Load(), tlsContact.Load())
	}
	t.Log(actual)
}

const csharpTestOnlyTransport = `namespace Rt;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;
/// TEST ONLY: this does not implement production lib.http.do.
public static class TestOnlyTransport{
 public static GoContext context;
 public static int cleaned,retained;
 static ManualResetEventSlim started;
 public static void sleep(GoTask t,long d){
  if(d==124){
   var gate=started;var token=R.sched.registerHost(t,()=>{});
   R.sched.launchHost(token,()=>Task.Run(()=>{if(!gate.Wait(10000))throw new Exception("server/native start barrier");return Array.Empty<object>();}));return;
  }
  bool cancelMode=d==126,rejection=d==129;
  string url=d==123?"HTTP_URL/normal":d==125?"HTTP_URL/release":d==126?"HTTP_URL/cancel":"TLS_URL";
  if(d==123||d==126)started=new ManualResetEventSlim();var startedGate=started;
  var cts=new CancellationTokenSource();Action cancel=()=>cts.Cancel();
  var sourceContext=context;Action detach=cancelMode?R.onCancel(context,cancel):()=>{};
  var token2=R.sched.registerHost(t,cancel,detach,()=>cancelMode&&sourceContext.err!=null?Array.Empty<object>():null);
  byte[] original={0,255,128,3};var input=(byte[])R.snapshot(original);original[1]=1;
  Interlocked.Increment(ref retained);
  // Native async completion owns copies/resources and only publishes mailbox records.
  _=run(token2,cts,input,url,d,rejection,startedGate);
 }
 static async Task run(HostToken token,CancellationTokenSource cts,byte[] input,string url,long d,bool rejection,ManualResetEventSlim gate){
  HostFault fault=null;HttpClient client=null;HttpResponseMessage response=null;System.IO.Stream body=null;SocketsHttpHandler handler=null;
  try{
   if(input[1]!=255)throw new Exception("input snapshot");handler=new SocketsHttpHandler();
   if(url.StartsWith("https:")&&!rejection)handler.SslOptions.RemoteCertificateValidationCallback=(sender,cert,chain,error)=>cert!=null&&Convert.ToBase64String(cert.GetRawCertData())=="TLS_CERT";
   client=new HttpClient(handler);client.Timeout=TimeSpan.FromSeconds(10);
   response=await client.GetAsync(url,HttpCompletionOption.ResponseHeadersRead,cts.Token).ConfigureAwait(false);
   if(rejection)throw new Exception("untrusted TLS accepted");
   body=await response.Content.ReadAsStreamAsync(cts.Token).ConfigureAwait(false);
   byte[] first=new byte[d==125?8:4];await body.ReadExactlyAsync(first,cts.Token).ConfigureAwait(false);
   if(d!=125){for(int i=0;i<4;i++)if(first[i]!=input[i])throw new Exception("independent binary bytes");}
   if(d==123||d==126)gate.Set();
   byte[] tail=new byte[16];while(await body.ReadAsync(tail,cts.Token).ConfigureAwait(false)!=0){}
  }
  catch(OperationCanceledException){if(d!=126&&!cts.IsCancellationRequested)fault=new HostFault("unexpected transport cancellation");}
  catch(HttpRequestException e){if(!rejection)fault=new HostFault("test transport: "+e);}
  catch(Exception e){fault=new HostFault("test-only adapter: "+e);}
  finally{
   if(body!=null)await body.DisposeAsync().ConfigureAwait(false);response?.Dispose();client?.Dispose();handler?.Dispose();cts.Dispose();
   input=null;body=null;response=null;client=null;handler=null;
   Interlocked.Decrement(ref retained);Interlocked.Increment(ref cleaned);token.complete(Array.Empty<object>(),fault);token.acknowledgeCleanup();
  }
 }
}`
