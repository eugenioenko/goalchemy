package contracts

import (
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

// Native runtime tests deliberately force collection at ownership boundaries;
// keeping a Swift ARC reference alone must not substitute for a traced Go root.
func TestSwiftRuntimeLifetime(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(root, "targets/swift"), filepath.Join(dir, "rt")); err != nil {
		t.Fatal(err)
	}
	source := `import Foundation
import GoalchemyNative
func require(_ value: Bool, _ text: String) { if !value { fatalError(text) } }
GTypes.table = [
 GType("int","int",64,true,-1,-1,0,[],[],[],[],true),
 GType("uint8","int",8,false,-1,-1,0,[],[],[],[],true),
 GType("State","struct",0,false,-1,-1,0,[0],["Value"],[false],[],true),
 GType("struct{}","struct",0,false,-1,-1,0,[],[],[],[],true),
 GType("string","string",0,false,-1,-1,0,[],[],[],[],true)
]
func cycleTest() {
 weak var discarded: GCell?
 do { let cell=GCell(.nilValue);cell.value = .pointer(cell);discarded=cell }
 require(discarded != nil,"cycle was not created")
 GHeap.collect()
 require(discarded == nil,"unreachable source cycle leaked")
 let cell=GCell(.nilValue);cell.value = .pointer(cell)
 GHeap.collect([.pointer(cell)])
 guard case .pointer(let alias)=cell.value else{fatalError("reachable cycle cleared")}
 require(alias===cell,"cycle identity changed")
 cell.value = .nilValue
}
func interiorTest() throws {
 let aggregate=GAggregate(2);aggregate.fields[0].value = .int(9)
 let field=aggregate.fields[0]
 GHeap.collect([.pointer(field)])
 require(field.value.intValue==9,"interior field root cleared")
 let buffer=GBuffer(bytes:[255,0,128],elem:1),byte=buffer.cell(2)
 GHeap.collect([.pointer(byte)])
 require(byte.value.unsigned==128 && buffer.count==3,"native byte proxy lost backing")
 byte.value = .integer(7,8,false);require(buffer.read(2).unsigned==7,"byte proxy no longer aliases backing")
}
func boundTest() throws {
 let receiver=GAggregate(2);receiver.fields[0].value = .int(17)
 GTypes.table[2].methods["Value"]=GFunction(1,{args,_ in GFrame.sync { [try GAggregate.field(args[0],0).value] }})
 let bound=try GInterface.bound(.interface(GInterface(2,.aggregate(receiver))),"Value")
 GHeap.collect([bound])
 let result=try GOwner(host:false).run(GFunction.start(bound,[]))
 require(result[0].intValue==17,"bound interface receiver or method root cleared")
 let captured=GCell(.int(23)),fn=GFunction(2,{_,env in GFrame.sync{[env[0].value]}},[captured])
 GHeap.collect([.function(fn)])
 require(try GOwner(host:false).run(fn.start([]))[0].intValue==23,"closure environment cleared")
}
func timerTest() throws {
 let owner=GOwner(host:false),frame=GFrame(-1);var done:GValue = .nilValue
 frame.roots={ [done] }
 frame.step={f,t in
  if f.pc==0 {
   let c=GContext(3);c.owner=owner
   c.timer=owner.timer(10,retained:[.opaque(c)]){c.cancel(true)}
   done = .channel(c.done);f.pc=1;owner.nextCollection=0;return .yield
  }
  if f.pc==1 {f.pc=2;return try GChannel.receive(done,t)}
  return .complete(t.rv)
 }
 let result=try owner.run(frame)
 require(result.count==2 && !result[1].boolValue,"timer-only context root lost blocked receiver")
}
func repeatedTest() throws {
 GHeap.collect();let before=GHeap.live
 let owner=GOwner(host:false),frame=GFrame(-1);var count=0
 frame.step={_,_ in
  owner.safepoint()
  if count%128==0 { require(GHeap.live<before+5000,"source cycles grew without bounded loop collections") }
  let c=GCell(.nilValue);c.value = .pointer(c);count+=1
  if count==12000{return .complete([])};return .yield
 }
 _ = try owner.run(frame);GHeap.collect()
 require(GHeap.live<=before+8,"loop safepoints failed to reclaim source cycles")
}
func keyTest() throws {
 let owner=GOwner(host:true);weak var released:GCryptoKey?
 do {
  let values=GCrypto.decode("generate_p256",GCrypto.work(GCryptoInput(operation:"generate_p256")),owner)
  released=try GCrypto.key(values[0])
 }
 require(released==nil,"owner retained an unreachable native key")
 let values=GCrypto.decode("generate_p256",GCrypto.work(GCryptoInput(operation:"generate_p256")),owner)
 let key=try GCrypto.key(values[0]);let lease=try key.snapshot()
 key.beginClose()
 do{_ = try key.snapshot();fatalError("Close failed to invalidate aliases immediately")}catch{}
 let finished=DispatchSemaphore(value:0)
 DispatchQueue.global().async{key.close();finished.signal()}
 require(finished.wait(timeout:.now()+0.01) == .timedOut,"Close did not wait for native read lease")
 key.release(lease);require(finished.wait(timeout:.now()+1) == .success,"Close did not acknowledge native cleanup")
 key.close()
 var wrapper:GoalchemyKey?
 weak var publicCell:GCell?
 weak var publicNative:GCryptoKey?
 do {
  let values=GCrypto.decode("generate_p256",GCrypto.work(GCryptoInput(operation:"generate_p256")),owner)
  guard case .pointer(let cell)=values[0] else{fatalError("key cell")}
  let native=try GCrypto.key(values[0]);publicCell=cell;publicNative=native
  wrapper=GoalchemyKey(native,cell)
 }
 GHeap.collect()
 require(publicCell != nil && publicNative != nil,"public wrapper root did not retain its native key cell")
 require(try GCrypto.key(.pointer(wrapper!.cell)) === publicNative!,"collector cleared externally rooted key cell")
 require(try !GEqual(.pointer(wrapper!.cell),.pointer(GCell(.opaque(publicNative!)))),"ordinary pointers compared by contents")
 wrapper=nil;GHeap.collect()
 require(publicCell==nil && publicNative==nil,"released wrapper leaked its source cell or native key")
}
try cycleTest();try interiorTest();try boundTest();try timerTest();try repeatedTest();try keyTest()
let bytes:[UInt8]=[255,0,128]
let error=try GNative.invoke("std.errors.new",[.string(bytes)])[0]
GHeap.collect([error])
require(try GOwner(host:false).run(GInterface.start(error,"Error",[]))[0].bytes==bytes,"native errors lost arbitrary string bytes")
require(try GEqual(error,error),"native error identity changed")
print("PASS Swift runtime lifetime")
`
	if err := os.WriteFile(filepath.Join(dir, "main.swift"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", `cc -Wno-deprecated-declarations -O2 -c rt/native/GoalchemyNative.c -o native.o $(pkg-config --cflags openssl zlib libcurl) && swiftc -swift-version 5 -suppress-warnings -I rt/native main.swift rt/types/*.swift rt/runtime/*.swift native.o $(pkg-config --libs openssl zlib libcurl) -o probe && ./probe`)
	cmd.Dir = dir
	cmd.Env = driver.ToolEnv()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native lifetime regression: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "PASS Swift runtime lifetime") {
		t.Fatal(string(output))
	}
}

// Panic and Error printing must preserve arbitrary Go bytes, including NUL and
// invalid UTF-8; Swift display strings cannot be used as runtime storage.
func TestSwiftErrorBytesNativeOracle(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	for name, text := range map[string]string{
		"go.mod": fmt.Sprintf("module swifterrors\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %q\n", root),
		"main.go": `package main
import "github.com/eugenioenko/goalchemy/lib/errors"
func main(){value:=string([]byte{255,0,128});err:=errors.New(value);println(err.Error());panic(err)}
`,
	} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	testutil.RunFixture(t, testutil.Fixture{Name: "error_bytes", Dir: source, Gate: "cooperative"}, []string{"go", "swift"})
}
