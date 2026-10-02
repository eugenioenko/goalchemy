package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eugenioenko/goalchemy/internal/testutil"
)

// Actual compiler frames use the new portable Promise artifact. The copied
// generated Sleep implementation is instrumented ONLY for this test to turn
// one sentinel duration into a genuine async transport suspension. No catalog
// capability is added and production lib.http.do remains unavailable.
func TestGeneratedTypeScriptHostOperations(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		time.Sleep(20 * time.Millisecond)
		w.Write([]byte{0, 255, 128, 3})
	}))
	defer srv.Close()
	tls := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("tls")) }))
	defer tls.Close()
	fixture, out := t.TempDir(), t.TempDir()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	mod := fmt.Sprintf("module tshostprobe\n\ngo 1.25\n\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root)
	source := `package main
import (
 "github.com/eugenioenko/goalchemy/lib/context"
 "github.com/eugenioenko/goalchemy/lib/time"
 "github.com/eugenioenko/goalchemy/lib/runtime"
)
func init(){println("source init")}
func main(){
 var missing context.Context
 println(missing==nil, nil==missing)
 parent,stop:=context.WithTimeout(context.Background(),5*time.Millisecond)
 defer stop()
 child,cancel:=context.WithTimeout(parent,time.Second)
 defer cancel()
 go func(){ runtime.Gosched(); println("progress") }()
 time.Sleep(123)
 println("transport returned")
 time.Sleep(10*time.Millisecond)
 println(parent.Err()==context.DeadlineExceeded,child.Err()==context.DeadlineExceeded)
 expired,ce:=context.WithTimeout(parent,time.Second)
 defer ce()
 println(expired.Err()==context.DeadlineExceeded)
}`
	for name, data := range map[string]string{"go.mod": mod, "main.go": source} {
		if err := os.WriteFile(filepath.Join(fixture, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if ds := testutil.CompileGate(fixture, "typescript", out, "cooperative"); len(ds) > 0 {
		t.Fatal(ds)
	}
	sleep := filepath.Join(out, "rt/runtime/std_time_sleep.ts")
	data, err := os.ReadFile(sleep)
	if err != nil {
		t.Fatal(err)
	}
	injection := `if(d===123n){
 const token=sched.registerHost(t,()=>{});
 sched.launchHost(token,async()=>{
  const r=await fetch(` + fmt.Sprintf("%q", srv.URL) + `); const bytes=new Uint8Array(await r.arrayBuffer());
  if(Array.from(bytes).join(',')!=='0,255,128,3')throw new Error('test-only echo mismatch');return [];
 });return;
}
`
	if strings.Count(string(data), "  t.rv = [];") != 1 {
		t.Fatal("test-only sentinel Sleep injection boundary changed")
	}
	data = []byte(strings.Replace(string(data), "  t.rv = [];", "  "+injection+"  t.rv = [];", 1))
	if err := os.WriteFile(sleep, data, 0600); err != nil {
		t.Fatal(err)
	}
	harness := `import { strict as assert } from 'node:assert';
import { request } from 'node:https';
import { X509Certificate } from 'node:crypto';
import { runMainHost, sched, sync, Frame, ret } from './rt/runtime/task_spawn.ts';
import { portableHost } from './rt/types/host.ts';
const output=[];const host={...portableHost,stdout:b=>output.push(...b),stderr:b=>output.push(...b),fail:n=>{throw new Error('exit '+n)}};
const {runHost}=await import('./host.ts');assert.equal(output.length,0,'import must not execute source');
const start=performance.now();await runHost(host);assert.ok(performance.now()-start>=20,'real source sleep');
const text=String.fromCharCode(...output);assert.equal(text,'source init\ntrue true\nprogress\ntransport returned\ntrue true\ntrue\n');
assert.equal(sched.tasks.size,0);assert.equal(sched.operations.size,0);
// Independent TLS server with explicit trust, visibly test-only Node adapter.
const certificate=new X509Certificate(Buffer.from(` + fmt.Sprintf("%q", fmt.Sprintf("%x", tls.Certificate().Raw)) + `,'hex')).toString();
class TLSFrame extends Frame {step(t){if(this.pc++===0){const token=sched.registerHost(t,()=>{});sched.launchHost(token,()=>new Promise((resolve,reject)=>{const req=request(` + fmt.Sprintf("%q", tls.URL) + `,{ca:certificate},res=>{const chunks=[];res.on('data',b=>chunks.push(b));res.on('end',()=>resolve([Buffer.concat(chunks).toString()]));});req.on('error',reject);req.end();}));}else{assert.equal(t.rv[0],'tls');ret(t,this);}}}
await runMainHost(new TLSFrame(),host);
console.log('PASS emitted cooperative frames/Promise artifact/import/real sleep/context/nil/source progress/HTTP/TLS');
`
	if err := os.WriteFile(filepath.Join(out, "test-only-host.ts"), []byte(harness), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "test-only-host.ts")
	cmd.Dir = out
	result, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("emitted TS host failed: %v\n%s", err, result)
	}
	if requests.Load() != 1 {
		t.Fatalf("Node did not contact independent HTTP server: %d", requests.Load())
	}
	t.Log(string(result))
	// Same compiler output and sentinel pause adapter in an actual browser.
	browserHarness := `import {runHost} from './host.ts';
import {portableHost} from './rt/types/host.ts';
export async function runEmitted(){
 const output:number[]=[];
 const host={...portableHost,stdout:(b:Uint8Array)=>output.push(...b),stderr:(b:Uint8Array)=>output.push(...b),fail:(n:number):never=>{throw new Error('exit '+n)}};
 if(output.length!==0)throw new Error('import executed source');
 const start=performance.now();await runHost(host);
 if(performance.now()-start<20)throw new Error('source sleep fast-forward');
 const text=String.fromCharCode(...output);
 if(text!=='source init\ntrue true\nprogress\ntransport returned\ntrue true\ntrue\n')throw new Error('emitted browser output '+JSON.stringify(text));
}`
	if err := os.WriteFile(filepath.Join(out, "browser-test-only.ts"), []byte(browserHarness), 0600); err != nil {
		t.Fatal(err)
	}
	browserCmd := exec.CommandContext(ctx, "node", "targets/typescript/tests/host_operations_browser.mjs", out)
	browserCmd.Dir = root
	browserResult, err := browserCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("actual emitted browser failed: %v\n%s", err, browserResult)
	}
	if requests.Load() != 2 {
		t.Fatalf("browser did not contact independent HTTP server: %d", requests.Load())
	}
	t.Log(string(browserResult))
}
