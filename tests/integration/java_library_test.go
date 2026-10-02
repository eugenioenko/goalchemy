package integration

import (
	"context"
	"fmt"
	"github.com/eugenioenko/goalchemy/internal/driver"
	"github.com/eugenioenko/goalchemy/internal/testutil"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGeneratedJavaImportingLibrary(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source, out, consumer := t.TempDir(), t.TempDir(), t.TempDir()
	fixture := `package probe
import("github.com/eugenioenko/goalchemy/lib/context";"github.com/eugenioenko/goalchemy/lib/callback";"github.com/eugenioenko/goalchemy/lib/crypto";"github.com/eugenioenko/goalchemy/lib/http";"github.com/eugenioenko/goalchemy/lib/time")
type Item struct {Name string;Data []byte}
type Value struct {Data []byte;Nested [][]byte;Names []string;Count int64;Small int8;Empty []byte;Item Item}
type Failure struct {Code string;Item Item;Items []Item}
func(e *Failure)Error()string{return e.Code}
var count int64=5
var fixed=[3]byte{0,255,128}
var failure=Failure{Code:"saved",Item:Item{Name:"old",Data:[]byte{9}},Items:[]Item{{Name:"nested",Data:[]byte{8}}}}
func Fixed(ctx context.Context)([]byte,error){return fixed[:],nil}
func ErrorGlobal(ctx context.Context)([]byte,error){return nil,&failure}
func Echo(ctx context.Context,v Value)(Value,error){v.Count=count;count++;if len(v.Data)>0{v.Data[0]++};return v,nil}
func Wide(ctx context.Context,v int64)(int64,error){return v,nil}
func Owned(ctx context.Context,v Value)(Value,error){_,e:=callback.Request(ctx,"held",nil);return v,e}
func FieldAlias(ctx context.Context,v Value)(Value,error){a:=&v.Item.Name;b:=&v.Item.Name;*a="alias";if a!=b{panic("noncanonical field")};_,e:=callback.Request(ctx,"held",nil);return v,e}
func Provider(ctx context.Context,name string,input []byte)([]byte,error){return callback.Request(ctx,name,input)}
func Fetch(ctx context.Context,url string,headers []string,max int,timeout int64)([]byte,error){_,_,b,e:=http.Do(ctx,"GET",url,headers,nil,max,timeout);return b,e}
func FetchHeaders(ctx context.Context,url string)([]string,error){_,h,_,e:=http.Do(ctx,"GET",url,nil,nil,1024,2000);return h,e}
func SourcePanic(ctx context.Context)([]byte,error){panic("secret source panic")}
func KeyProvider(ctx context.Context,name string)([]byte,error){key,e:=crypto.GenerateP256();if e!=nil{return nil,e};defer key.Close();return callback.Request(ctx,name,nil)}
func KeySuccess(ctx context.Context)([]byte,error){key,e:=crypto.GenerateP256();if e!=nil{return nil,e};defer key.Close();return []byte{7},nil}
func GoClose(ctx context.Context)([]byte,error){key,e:=crypto.GenerateP256();if e!=nil{return nil,e};go key.Close();time.Sleep(time.Millisecond);_,e=key.PublicPEM();if e==nil{return nil,&Failure{Code:"go close failed"}};return []byte{8},nil}
func BoundClose(ctx context.Context)([]byte,error){key,e:=crypto.GenerateP256();if e!=nil{return nil,e};close:=key.Close;defer close();return []byte{9},nil}
func KeyRace(ctx context.Context)([]byte,error){key,e:=crypto.GenerateRSA2048();if e!=nil{return nil,e};alias:=key;ch:=make(chan int,1);go func(){sig,e:=crypto.RS256Sign(key,[]byte{0,255});if e!=nil{ch<-0}else{ch<-len(sig)}}();_,e=callback.Request(ctx,"ready",nil);if e!=nil{return nil,e};go alias.Close();time.Sleep(time.Millisecond);_,e=key.PublicPEM();if e==nil{return nil,&Failure{Code:"alias open"}};_,e=callback.Request(ctx,"closed",nil);if e!=nil{return nil,e};n:=<-ch;if n!=256{return nil,&Failure{Code:"snapshot failed"}};return []byte{1,2,255},nil}
func AcquireFault(ctx context.Context)([]byte,error){go func(){key,e:=crypto.GenerateRSA2048();if e==nil{key.Close()}}();time.Sleep(time.Millisecond);return callback.Request(ctx,"fault",nil)}
func Derive(ctx context.Context)([]byte,error){return crypto.HKDFSHA256([]byte{11},nil,nil,32)}
`
	mod := fmt.Sprintf("module libraryprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root)
	for n, data := range map[string]string{"go.mod": mod, "library.go": fixture} {
		if e := os.WriteFile(filepath.Join(source, n), []byte(data), 0600); e != nil {
			t.Fatal(e)
		}
	}
	if ds := testutil.CompileGate(source, "java", out, "cooperative"); len(ds) > 0 {
		t.Fatal(ds)
	}
	generated, e := os.ReadFile(filepath.Join(out, "Generated.java"))
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(generated), "package io.goalchemy.generated;") || !strings.Contains(string(generated), "Library.submit(") {
		t.Fatal("missing native exports")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	env := driver.ToolEnv()
	bin := ""
	for _, v := range env {
		if strings.HasPrefix(v, "JAVA_HOME=") {
			bin = filepath.Join(strings.TrimPrefix(v, "JAVA_HOME="), "bin")
		}
	}
	build := exec.CommandContext(ctx, "sh", "build.sh")
	build.Dir = out
	build.Env = env
	if data, e := build.CombinedOutput(); e != nil {
		t.Fatalf("generated JAR: %v\n%s", e, data)
	}
	suite, e := os.ReadFile(filepath.Join(root, "targets/java/tests/LibraryTest.java.in"))
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(consumer, "LibraryTest.java"), suite, 0600); e != nil {
		t.Fatal(e)
	}
	deps := exec.CommandContext(ctx, "bash", filepath.Join(root, "targets/java/tests/crypto-dependencies.sh"))
	deps.Env = env
	data, e := deps.Output()
	if e != nil {
		t.Fatal(e)
	}
	jar := strings.TrimSpace(string(data))
	cp := filepath.Join(out, "goalchemy-generated.jar")
	cmd := exec.CommandContext(ctx, filepath.Join(bin, "javac"), "-encoding", "UTF-8", "-cp", cp, "-d", consumer, filepath.Join(consumer, "LibraryTest.java"))
	cmd.Env = env
	if data, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("separate named consumer: %v\n%s", e, data)
	}
	for _, mode := range []string{"missing-dependency", "all"} {
		testCP := consumer + ":" + cp
		if mode == "all" {
			testCP += ":" + jar
		}
		args := []string{"-Djava.security.manager=allow", "-cp", testCP, "consumer.LibraryTest", mode}
		cmd = exec.CommandContext(ctx, filepath.Join(bin, "java"), args...)
		cmd.Env = env
		if data, e := cmd.CombinedOutput(); e != nil {
			t.Fatalf("actual importing consumer %s: %v\n%s", mode, e, data)
		} else {
			t.Log(string(data))
		}
	}
}

// Exercises the shared receiver wrapper repair on the three production library targets.
func TestNativeCapabilityMethodValues(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	fixture := `package main
import("github.com/eugenioenko/goalchemy/lib/crypto";"github.com/eugenioenko/goalchemy/lib/time")
type Holder struct{*crypto.Key}
func makeKey()*crypto.Key{generate:=crypto.GenerateP256;k,e:=generate();if e!=nil{panic(e)};return k}
func deferred(k *crypto.Key){close:=k.Close;defer close()}
func main(){
 k:=makeKey();public:=k.PublicPEM;p,e:=public();if e!=nil||len(p)==0{panic("bound public")}
 pub:=(*crypto.Key).PublicPEM;p,e=pub(k);if e!=nil||len(p)==0{panic("expression public")}
 holder:=Holder{Key:k};promoted:=holder.PublicPEM;p,e=promoted();if e!=nil||len(p)==0{panic("promoted public")}
 promotedExpr:=(*Holder).PublicPEM;p,e=promotedExpr(&holder);if e!=nil||len(p)==0{panic("promoted expression")}
 close:=k.Close;close();_,e=public();if e==nil{panic("bound close")}
 k=makeKey();deferred(k);_,e=k.PublicPEM();if e==nil{panic("deferred bound close")}
 k=makeKey();closeExpr:=(*crypto.Key).Close;go closeExpr(k);time.Sleep(time.Millisecond);_,e=k.PublicPEM();if e==nil{panic("go expression close")}
 k=makeKey();goClose:=k.Close;go goClose();time.Sleep(time.Millisecond);_,e=k.PublicPEM();if e==nil{panic("go bound close")}
 println("native-bound-methods-ok")
}`
	mod := fmt.Sprintf("module methodprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root)
	for name, data := range map[string]string{"go.mod": mod, "main.go": fixture} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, target := range []string{"go", "typescript", "java"} {
		t.Run(target, func(t *testing.T) {
			out := t.TempDir()
			if ds := testutil.CompileGate(source, target, out, "cooperative"); len(ds) > 0 {
				t.Fatal(ds)
			}
			if target == "go" {
				result, err := testutil.Runners[target](out)
				if err != nil || result.Exit != 0 || !strings.Contains(result.Stdout+result.Stderr, "native-bound-methods-ok") {
					t.Fatalf("actual native methods: %v %+v", err, result)
				}
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			env := driver.ToolEnv()
			var cmd *exec.Cmd
			if target == "typescript" {
				if err := os.WriteFile(filepath.Join(out, "probe.ts"), []byte("import './rt/types/node_host.ts';import {$runHost} from './program.ts';await $runHost();\n"), 0600); err != nil {
					t.Fatal(err)
				}
				cmd = exec.CommandContext(ctx, "node", "--stack-size=4000", "probe.ts")
			} else {
				if err := os.WriteFile(filepath.Join(out, "HostRunner.java"), []byte("public class HostRunner { public static void main(String[] args){Main.runHost();} }"), 0600); err != nil {
					t.Fatal(err)
				}
				var sources []string
				filepath.WalkDir(out, func(p string, d fs.DirEntry, e error) error {
					if e != nil {
						return e
					}
					if !d.IsDir() && strings.HasSuffix(p, ".java") {
						sources = append(sources, p)
					}
					return nil
				})
				javaBin := ""
				for _, item := range env {
					if strings.HasPrefix(item, "JAVA_HOME=") {
						javaBin = filepath.Join(strings.TrimPrefix(item, "JAVA_HOME="), "bin")
					}
				}
				build := exec.CommandContext(ctx, filepath.Join(javaBin, "javac"), append([]string{"-d", out}, sources...)...)
				build.Env = env
				if data, e := build.CombinedOutput(); e != nil {
					t.Fatalf("native host compilation: %v\n%s", e, data)
				}
				cmd = exec.CommandContext(ctx, filepath.Join(javaBin, "java"), "-cp", out, "HostRunner")
			}
			cmd.Dir = out
			cmd.Env = env
			if data, e := cmd.CombinedOutput(); e != nil || !strings.Contains(string(data), "native-bound-methods-ok") {
				t.Fatalf("actual native methods: %v\n%s", e, data)
			}
		})
	}
}
