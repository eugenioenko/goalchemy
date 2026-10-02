package integration

import (
	"fmt"
	"github.com/eugenioenko/goalchemy/internal/testutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedGoImportingLibrary(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source, out, consumer := t.TempDir(), t.TempDir(), t.TempDir()
	mod := fmt.Sprintf("module libraryprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root)
	fixture := `package probe
import (
 "github.com/eugenioenko/goalchemy/lib/context"
 "github.com/eugenioenko/goalchemy/lib/callback"
 "github.com/eugenioenko/goalchemy/lib/http"
 "github.com/eugenioenko/goalchemy/lib/crypto"
)
type Value struct { Data []byte; Nested [][]byte; Count int64; Empty []byte }
var count int64 = 5
var fixed = [3]byte{0,255,128}
type Failure struct { Code string; Obligations []string }
func (e *Failure) Error()string{return e.Code}
var failure=Failure{Code:"saved",Obligations:[]string{"obligation"}}
func Fixed(ctx context.Context)([]byte,error){return fixed[:],nil}
func ErrorGlobal(ctx context.Context)([]byte,error){return nil,&failure}
func Echo(ctx context.Context, value Value)(Value,error){count++;value.Count=count;if len(value.Data)>0 {value.Data[0]++};return value,nil}
func Provider(ctx context.Context,name string,request []byte)([]byte,error){return callback.Request(ctx,name,request)}
func Fetch(ctx context.Context,url string)([]byte,error){_,_,b,e:=http.Do(ctx,"GET",url,nil,nil,1024,2000);return b,e}
func SourcePanic(ctx context.Context)([]byte,error){panic("secret source panic")}
func KeyProvider(ctx context.Context,name string)([]byte,error){key,e:=crypto.GenerateP256();if e!=nil{return nil,e};defer key.Close();return callback.Request(ctx,name,nil)}
`
	for name, data := range map[string]string{"go.mod": mod, "library.go": fixture} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if ds := testutil.CompileGate(source, "go", out, "cooperative"); len(ds) > 0 {
		t.Fatal(ds)
	}
	generated, err := os.ReadFile(filepath.Join(out, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "package generated") || !strings.Contains(string(generated), "rt.RunLibrary(") {
		t.Fatal("missing generated library boundary")
	}
	consumerMod := fmt.Sprintf("module independent\n\ngo 1.25\nrequire goalchemyout v0.0.0\nreplace goalchemyout => %s\n", out)
	test := `package independent
import (
 "bytes"
 "context"
 "errors"
 "net/http"
 "net/http/httptest"
 "sync"
 "testing"
 "time"
 g "goalchemyout"
)
func TestValueAndInit(t *testing.T){
 input:=g.Value{Data:[]byte{0,255},Nested:[][]byte{[]byte{9}},Empty:[]byte{}}
 first,e:=g.Echo(context.Background(),input);if e!=nil || first.Count!=6 || first.Data[0]!=1 || input.Data[0]!=0 || first.Empty==nil {t.Fatal("snapshot/init",e)}
 second,e:=g.Echo(context.Background(),g.Value{});if e!=nil || second.Count!=6 || second.Data!=nil || first.Data[0]!=1 {t.Fatal("fresh initialization/results",e)}
 first.Nested[0][0]=1;if input.Nested[0][0]!=9{t.Fatal("nested ownership")}
}
func TestGlobalReturnOwnership(t *testing.T){
 b,e:=g.Fixed(context.Background());if e!=nil || !bytes.Equal(b,[]byte{0,255,128}){t.Fatal("global array reset before copying",e,b)}
 _,e=g.ErrorGlobal(context.Background());var source *g.Failure;if !errors.As(e,&source) || source.Code!="saved" || len(source.Obligations)!=1 || source.Obligations[0]!="obligation"{t.Fatal("source error reset before copying",e)}
 var wg sync.WaitGroup;for i:=0;i<12;i++{wg.Add(1);go func(){defer wg.Done();v,e:=g.Fixed(context.Background());if e!=nil || !bytes.Equal(v,[]byte{0,255,128}){t.Error("global result ownership",e,v)}}()};wg.Wait()
 if !bytes.Equal(b,[]byte{0,255,128}) || source.Code!="saved" || source.Obligations[0]!="obligation"{t.Fatal("retained results changed")}
}
func TestProviderIsolationFaultsAndCancellation(t *testing.T){
 callbacks:=g.Callbacks{"token":func(ctx context.Context,input []byte,settle func([]byte,error))func(){reply:=[]byte{255,0};settle(reply,nil);reply[0]=0;settle([]byte{1},nil);return nil}}
 b,e:=g.Provider(context.Background(),"token",[]byte{0,255},callbacks);if e!=nil || !bytes.Equal(b,[]byte{255,0}){t.Fatal("provider",e)}
 b,e=g.Provider(context.Background(),"token",nil);if e==nil || b!=nil{t.Fatal("registry survived")}
 rejection:=errors.New("reject");callbacks["token"]=func(ctx context.Context,b []byte,settle func([]byte,error))func(){settle([]byte{1},rejection);return nil}
 b,e=g.Provider(context.Background(),"token",nil,callbacks);if !errors.Is(e,rejection) || b!=nil{t.Fatal("rejection")}
 callbacks["token"]=func(ctx context.Context,b []byte,settle func([]byte,error))func(){panic("secret fault")}
 b,e=g.KeyProvider(context.Background(),"token",callbacks);var failure *g.LibraryError;if b!=nil || !errors.As(e,&failure) || failure.Kind!="host_fault"{t.Fatal("fault",e)}
 b,e=g.SourcePanic(context.Background());if b!=nil || !errors.As(e,&failure) || failure.Kind!="source_panic"{t.Fatal("source panic",e)}
 started:=make(chan struct{});release:=make(chan struct{});finished:=make(chan error,1);var late func([]byte,error)
 ctx,cancel:=context.WithCancel(context.Background())
 callbacks["token"]=func(ctx context.Context,b []byte,settle func([]byte,error))func(){late=settle;close(started);go func(){<-release;settle(nil,ctx.Err())}();return func(){close(release)}}
 go func(){_,e:=g.Provider(ctx,"token",nil,callbacks);finished<-e}();<-started
 queued,stop:=context.WithTimeout(context.Background(),20*time.Millisecond);defer stop()
 _,e=g.Echo(queued,g.Value{});if !errors.Is(e,context.DeadlineExceeded){t.Fatal("queued cancellation",e)}
 cancel();if !errors.Is(<-finished,context.Canceled){t.Fatal("active cancellation")}
 late([]byte{1},nil)
 _,e=g.Echo(context.Background(),g.Value{});if e!=nil{t.Fatal("reservation not released",e)}
 var wg sync.WaitGroup;for i:=0;i<12;i++{wg.Add(1);go func(){defer wg.Done();v,e:=g.Echo(context.Background(),g.Value{});if e!=nil || v.Count!=6{t.Error("overlapping isolation",e)}}()};wg.Wait()
}
func TestProviderStopFaultWaitsForResourceAcknowledgment(t *testing.T){
 started:=make(chan struct{}); stopCalled:=make(chan struct{}); release:=make(chan struct{}); result:=make(chan error,1)
 ctx,cancel:=context.WithCancel(context.Background())
 callbacks:=g.Callbacks{"lease":func(ctx context.Context,_ []byte,settle func([]byte,error))func(){
  close(started)
  go func(){<-release;settle(nil,ctx.Err())}()
  return func(){close(stopCalled);panic("stop hook fault")}
 }}
 go func(){_,err:=g.KeyProvider(ctx,"lease",callbacks);result<-err}()
 <-started;cancel();<-stopCalled
 select {case <-result:t.Fatal("fault acknowledged before provider released resources");case <-time.After(20*time.Millisecond):}
 close(release)
 var fault *g.LibraryError;if !errors.As(<-result,&fault) || fault.Kind!="host_fault"{t.Fatal("stop fault classification")}
 _,err:=g.Echo(context.Background(),g.Value{});if err!=nil{t.Fatal("owner release",err)}
}
func TestRealHostIO(t *testing.T){
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){if r.URL.Path=="/cancel"{w.Write([]byte{1});w.(http.Flusher).Flush();<-r.Context().Done();return};w.Write([]byte{0,255})}));defer server.Close()
 b,e:=g.Fetch(context.Background(),server.URL);if e!=nil || !bytes.Equal(b,[]byte{0,255}){t.Fatal("host I/O",e)}
 ctx,cancel:=context.WithTimeout(context.Background(),25*time.Millisecond);defer cancel();b,e=g.Fetch(ctx,server.URL+"/cancel");if b!=nil || !errors.Is(e,context.DeadlineExceeded){t.Fatal("partial/cancellation",e)}
}
`
	for name, data := range map[string]string{"go.mod": consumerMod, "consumer_test.go": test} {
		if err := os.WriteFile(filepath.Join(consumer, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "test", "-race", "-count=1", "./...")
	cmd.Dir = consumer
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=go1.25.14", "GOFLAGS=")
	result, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("independent generated library: %v\n%s", err, result)
	}
}

func TestGoLibraryRejectsUnsupportedPublicBoundary(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	fixture := t.TempDir()
	mod := fmt.Sprintf("module unsupportedlib\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root)
	if err := os.WriteFile(filepath.Join(fixture, "go.mod"), []byte(mod), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "library.go"), []byte("package unsupported\nfunc Borrow(value *int) int {return *value}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ds := testutil.CompileGate(fixture, "go", t.TempDir(), "cooperative")
	if len(ds) != 1 || ds[0].Code != "GCE007" {
		t.Fatal(ds)
	}
	for _, target := range []string{"typescript", "python", "java", "csharp", "rust"} {
		ds := testutil.CompileGate(fixture, target, t.TempDir(), "cooperative")
		if len(ds) != 1 || ds[0].Code != "GCE006" {
			t.Fatalf("%s: %v", target, ds)
		}
	}
}
