package integration

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/fs"
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
func TestGeneratedJavaHostOperations(t *testing.T) {
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
	for name, data := range map[string]string{"go.mod": fmt.Sprintf("module javahostprobe\n\ngo 1.25\n\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root), "main.go": source} {
		if err := os.WriteFile(filepath.Join(fixture, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if ds := testutil.CompileGate(fixture, "java", out, "cooperative"); len(ds) > 0 {
		t.Fatal(ds)
	}
	// Both source and virtual runtime are exercised unchanged before injection;
	// sentinel sleeps are ordinary short virtual sleeps in this first run.
	env := driver.ToolEnv()
	bin := ""
	for _, item := range env {
		if strings.HasPrefix(item, "JAVA_HOME=") {
			bin = filepath.Join(strings.TrimPrefix(item, "JAVA_HOME="), "bin")
		}
	}
	ctx, stop := context.WithTimeout(context.Background(), 120*time.Second)
	defer stop()
	compile := func() {
		t.Helper()
		var sources []string
		if err := filepath.WalkDir(out, func(path string, d fs.DirEntry, err error) error {
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
		cmd := exec.CommandContext(ctx, filepath.Join(bin, "javac"), append([]string{"-d", out}, sources...)...)
		cmd.Env = env
		if result, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("actual emitted Java compile: %v\n%s", err, result)
		}
	}
	run := func(class string) string {
		t.Helper()
		cmd := exec.CommandContext(ctx, filepath.Join(bin, "java"), "-ea", "-cp", out, class)
		cmd.Env = env
		result, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("actual emitted Java %s: %v\n%s", class, err, result)
		}
		return string(result)
	}
	compile()
	baseline := run("Main")
	if baseline != "init 1\nmain 2\ntrue\ntrue true\ntransport returned\nsource progress\ncancel progress\ntrue\ntrue true\n" {
		t.Fatalf("unchanged generated virtual behavior: %q", baseline)
	}
	inject := func(path, marker, replacement string) {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(data), marker) != 1 {
			t.Fatalf("test-only unique replacement marker changed: %s", path)
		}
		if err := os.WriteFile(path, []byte(strings.Replace(string(data), marker, replacement, 1)), 0600); err != nil {
			t.Fatal(err)
		}
	}
	inject(filepath.Join(out, "rt/runtime/StdTimeSleep.java"), "        t.rv = new Object[0];", `        if (d>=123 && d<=129) { TestOnlyTransport.sleep(t,d);return; }
        t.rv = new Object[0];`)
	inject(filepath.Join(out, "rt/runtime/StdContextWithCancel.java"), "        StdContextErr.Context c = StdContextErr.newChild(parent);", `        StdContextErr.Context c = StdContextErr.newChild(parent);
        TestOnlyTransport.context=c;`)
	adapter := strings.NewReplacer("HTTP_URL", srv.URL, "TLS_URL", tls.URL, "TLS_CERT", base64.StdEncoding.EncodeToString(tls.Certificate().Raw)).Replace(javaTestOnlyTransport)
	if err := os.WriteFile(filepath.Join(out, "rt/runtime/TestOnlyTransport.java"), []byte(adapter), 0600); err != nil {
		t.Fatal(err)
	}
	consumer := `import rt.*;
public final class HostConsumer {
 public static void main(String[] args) {
  Main.runHost(); Main.runHost();
  if(TestOnlyTransport.cleaned.get()!=10)throw new AssertionError("native cleanup count "+TestOnlyTransport.cleaned);
  if(TestOnlyTransport.retained.get()!=0)throw new AssertionError("native retained inputs/bodies");
  if(Program.panicState.get()==null)throw new AssertionError("panic binding after reuse");
  System.out.println("PASS emitted Java host frames/repeated init/HTTP/TLS/cancellation/cleanup");
 }
}`
	if err := os.WriteFile(filepath.Join(out, "HostConsumer.java"), []byte(consumer), 0600); err != nil {
		t.Fatal(err)
	}
	compile()
	actual := run("HostConsumer")
	want := "init 1\nmain 2\ntrue\ntrue true\nsource progress\ntransport returned\ncancel progress\ntrue\ntrue true\ninit 3\nmain 4\ntrue\ntrue true\nsource progress\ntransport returned\ncancel progress\ntrue\ntrue true\nPASS emitted Java host frames/repeated init/HTTP/TLS/cancellation/cleanup\n"
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

const javaTestOnlyTransport = `package rt;
import java.net.*;
import javax.net.ssl.*;
import java.io.*;
import java.security.*;
import java.security.cert.*;
import java.util.concurrent.*;
import java.util.concurrent.atomic.*;
/** TEST ONLY: no production HTTP contract or capability mapping. */
public final class TestOnlyTransport {
 public static StdContextErr.Context context;
 public static AtomicInteger cleaned=new AtomicInteger(),retained=new AtomicInteger();
 static CountDownLatch started;
 static SSLContext trust() throws Exception {
  var cert=CertificateFactory.getInstance("X.509").generateCertificate(new ByteArrayInputStream(java.util.Base64.getDecoder().decode("TLS_CERT")));
  var store=KeyStore.getInstance(KeyStore.getDefaultType());store.load(null,null);store.setCertificateEntry("test",cert);
  var factory=TrustManagerFactory.getInstance(TrustManagerFactory.getDefaultAlgorithm());factory.init(store);
  var ssl=SSLContext.getInstance("TLS");ssl.init(null,factory.getTrustManagers(),null);return ssl;
 }
 public static void sleep(TaskSpawn.Task t,long d) {
  if(d==124){
   var gate=started;var token=TaskSpawn.sched.registerHost(t,()->{});
   new Thread(()->{try{if(!gate.await(10,TimeUnit.SECONDS))throw new AssertionError("server/native start barrier");token.complete(new Object[0]);}catch(Throwable e){token.complete(new Object[0],new TaskSpawn.HostFault(String.valueOf(e)));}finally{token.acknowledgeCleanup();}}).start();return;
  }
  boolean cancelMode=d==126, rejection=d==129;
  String url=d==123?"HTTP_URL/normal":d==125?"HTTP_URL/release":d==126?"HTTP_URL/cancel":"TLS_URL";
  if(d==123||d==126)started=new CountDownLatch(1);
  var gate=started;
  var connection=new AtomicReference<HttpURLConnection>();var canceled=new AtomicBoolean();var closing=new CompletableFuture<Void>();
  Runnable cancel=()->{
   if(canceled.compareAndSet(false,true))new Thread(()->{try{var c=connection.get();if(c!=null)c.disconnect();}finally{closing.complete(null);}}).start();
  };
  Runnable detach=cancelMode?StdContextErr.onCancel(context,cancel):()->{};
  var sourceContext=context;
  var token=TaskSpawn.sched.registerHost(t,cancel,detach,()->cancelMode&&sourceContext.err!=null?new Object[0]:null);
  byte[] original={0,(byte)255,(byte)128,3};var input=new AtomicReference<byte[]>((byte[])TaskSpawn.snapshot(original));original[1]=1;
  retained.incrementAndGet();
  new Thread(()->{
   Object[] result=new Object[0];TaskSpawn.HostFault fault=null;HttpURLConnection c=null;InputStream body=null;
   try{
    if(input.get()[1]!=(byte)255)throw new AssertionError("input snapshot");
    c=(HttpURLConnection)new URL(url).openConnection();
    c.setConnectTimeout(10000);c.setReadTimeout(10000);
    if(c instanceof HttpsURLConnection https&&!rejection)https.setSSLSocketFactory(trust().getSocketFactory());
    connection.set(c);if(canceled.get())throw new IOException("already canceled");
    body=c.getInputStream();
    if(rejection)throw new AssertionError("untrusted TLS accepted");
    byte[] first=body.readNBytes(d==125?8:4);
    if(d!=125&&!java.util.Arrays.equals(first,input.get()))throw new AssertionError("independent server binary bytes");
    if(d==123||d==126)gate.countDown();
    while(body.read()!=-1){}
   }catch(SSLHandshakeException expected){if(!rejection)fault=new TaskSpawn.HostFault("trusted TLS rejected: "+expected);}
   catch(IOException expected){if(!cancelMode&&!canceled.get())fault=new TaskSpawn.HostFault("test transport failure: "+expected);}
   catch(Throwable e){fault=new TaskSpawn.HostFault("test-only adapter: "+e);}
   finally{
    try{if(body!=null)body.close();}catch(IOException ignored){}
    if(c!=null)c.disconnect();connection.set(null);
    if(canceled.get())closing.join();
    input.set(null);body=null;c=null;retained.decrementAndGet();cleaned.incrementAndGet();
    token.complete(result,fault);token.acknowledgeCleanup();
   }
  }).start();
 }
}
`
