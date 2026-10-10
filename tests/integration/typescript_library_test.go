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

func TestGeneratedTypeScriptImportingLibrary(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source, out := t.TempDir(), t.TempDir()
	mod := fmt.Sprintf("module libraryprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root)
	fixture := `package probe
import("github.com/eugenioenko/goalchemy/lib/context";"github.com/eugenioenko/goalchemy/lib/callback";"github.com/eugenioenko/goalchemy/lib/crypto";"github.com/eugenioenko/goalchemy/lib/http";"github.com/eugenioenko/goalchemy/lib/time")
type Value struct{Data []byte;Nested [][]byte;Names []string;Count int64;Empty []byte}
var count int64=5
var fixed=[3]byte{0,255,128}
type Failure struct{Code string;Obligations []string}
func(e *Failure)Error()string{return e.Code}
var failure=Failure{Code:"saved",Obligations:[]string{"obligation"}}
func Fixed(ctx context.Context)([]byte,error){return fixed[:],nil}
func ErrorGlobal(ctx context.Context)([]byte,error){return nil,&failure}
func Echo(ctx context.Context,value Value)(Value,error){count++;value.Count=count;if len(value.Data)>0{value.Data[0]++};return value,nil}
func Provider(ctx context.Context,name string,request []byte)([]byte,error){return callback.Request(ctx,name,request)}
func Fetch(ctx context.Context,url string)([]byte,error){_,_,b,e:=http.Do(ctx,"GET",url,nil,nil,1024,2000);return b,e}
func FetchHeaders(ctx context.Context,url string,headers []string)([]byte,error){_,_,b,e:=http.Do(ctx,"GET",url,headers,nil,1024,2000);return b,e}
func SourcePanic(ctx context.Context)([]byte,error){panic("secret source panic")}
func KeyProvider(ctx context.Context,name string)([]byte,error){key,e:=crypto.GenerateP256();if e!=nil{return nil,e};defer key.Close();return callback.Request(ctx,name,nil)}
func GoClose(ctx context.Context)([]byte,error){key,e:=crypto.GenerateP256();if e!=nil{return nil,e};go key.Close();time.Sleep(time.Millisecond);_,e=key.PublicPEM();if e==nil{return nil,&Failure{Code:"go close failed"}};return []byte{8},nil}
func KeySuccess(ctx context.Context)([]byte,error){key,e:=crypto.GenerateP256();if e!=nil{return nil,e};defer key.Close();return []byte{7},nil}
func AcquirePanic(ctx context.Context)([]byte,error){go func(){time.Sleep(time.Millisecond*2);panic("secret") }();key,e:=crypto.GenerateP256();if e!=nil{return nil,e};defer key.Close();return nil,nil}
func KeyRace(ctx context.Context)([]byte,error){key,e:=crypto.GenerateRSA2048();if e!=nil{return nil,e};alias:=key;ch:=make(chan int,1);go func(){sig,err:=crypto.RS256Sign(key,[]byte{0,255});if err!=nil{ch<-0}else{ch<-len(sig)}}();_,e=callback.Request(ctx,"ready",nil);if e!=nil{return nil,e};alias.Close();_,e=crypto.RS256Sign(key,nil);if e==nil{return nil,&Failure{Code:"alias remained open"}};n:=<-ch;if n!=256{return nil,&Failure{Code:"snapshot failed"}};return []byte{1,2,255},nil}
`
	for name, data := range map[string]string{"go.mod": mod, "library.go": fixture} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if ds := testutil.CompileGate(source, "typescript", out, "cooperative"); len(ds) > 0 {
		t.Fatal(ds)
	}
	generated, err := os.ReadFile(filepath.Join(out, "main.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "rt.runLibraryCall(") || strings.Contains(string(generated), "$run()") {
		t.Fatal("missing actual generated exports")
	}
	suite, err := os.ReadFile(filepath.Join(root, "targets/typescript/tests/library_suite.ts.in"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "library-suite.ts"), suite, 0600); err != nil {
		t.Fatal(err)
	}
	config := `{"compilerOptions":{"target":"ES2022","module":"NodeNext","moduleResolution":"NodeNext","rewriteRelativeImportExtensions":true,"declaration":true,"outDir":"dist","strict":true,"skipLibCheck":true,"lib":["ES2022","DOM","DOM.Iterable"]},"include":["*.ts","rt/**/*.ts"]}`
	if err := os.WriteFile(filepath.Join(out, "tsconfig.json"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("tsc", "-p", ".")
	build.Dir = out
	if data, err := build.CombinedOutput(); err != nil {
		t.Fatalf("native declarations/build: %v\n%s", err, data)
	}
	node := exec.Command("node", "--input-type=module", "-e", `const {createServer}=await import('node:http');const {runLibrarySuite}=await import('./dist/library-suite.js');const server=createServer((req,res)=>{if(req.url==='/truncated'){res.writeHead(200,{'Content-Length':'1000'});res.flushHeaders();res.write('{');setTimeout(()=>res.destroy(),50);return;}const allowed=req.url==='/fetch'||(decodeURI(req.url)==='/日本語?q=café'&&req.headers.authorization==='Bearer fixture'&&req.headers.dpop==='fixture-proof'&&req.headers['content-type']==='application/json'&&req.headers['connect-protocol-version']==='1'&&req.headers['x-copy']==='trimmed'&&req.headers['x-method-override']==='GET');res.end(Uint8Array.from(allowed?[0,255,128]:[1]));});await new Promise(r=>server.listen(0,'127.0.0.1',r));try{await runLibrarySuite('http://127.0.0.1:'+server.address().port+'/fetch',[values=>Buffer.from(values)]);}finally{server.closeAllConnections();await new Promise(r=>server.close(r));}`)
	node.Dir = out
	if data, err := node.CombinedOutput(); err != nil {
		t.Fatalf("importing consumer: %v\n%s", err, data)
	} else {
		t.Log(string(data))
	}
	browser := exec.Command("node", "targets/typescript/tests/library_browser.mjs", out)
	browser.Dir = root
	if data, err := browser.CombinedOutput(); err != nil {
		t.Fatalf("browser consumer: %v\n%s", err, data)
	} else {
		t.Log(string(data))
	}
}
