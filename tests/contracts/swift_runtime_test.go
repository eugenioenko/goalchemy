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

// Native error values have different Go method sets: ordinary errors implement
// Error only, runtime panics add RuntimeError, and the deadline sentinel adds
// Timeout and Temporary. Exercise assertions, dispatch, identity, and names.
func TestSwiftNativeErrorMethodsNativeOracle(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	for name, text := range map[string]string{
		"go.mod": fmt.Sprintf("module swifterrormethods\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %q\n", root),
		"main.go": `package main
import (
 "github.com/eugenioenko/goalchemy/lib/context"
 "github.com/eugenioenko/goalchemy/lib/crypto"
 "github.com/eugenioenko/goalchemy/lib/errors"
)
type RuntimeMarker interface { RuntimeError() }
type RuntimeError interface { Error() string; RuntimeError() }
type Timeout interface { Timeout() bool }
type Temporary interface { Temporary() bool }
type DeadlineError interface { Error() string; Timeout() bool; Temporary() bool }
type Message = string
type Flag = bool
type DeadlineAlias interface { Error() Message; Timeout() Flag; Temporary() Flag }
type Extra interface { Extra() }
type UserError struct{}
func (UserError) Error() string { return "user" }
func (UserError) RuntimeError() { println("user marker invoked") }
type Calculator struct{}
func (Calculator) Calculate(n int) int { return n + 1 }
type Word string
type NamedError struct{}
func (NamedError) Error() Word { return "named result" }
var unwrapCalls, isCalls int
type WrongResult struct{}
func (WrongResult) Error() string { return "wrong result" }
func (WrongResult) Unwrap() string { unwrapCalls++; return "wrong" }
func (WrongResult) Is(error) int { isCalls++; return 0 }
type WrongParam struct { inner error }
func (WrongParam) Error() string { return "wrong param" }
func (w WrongParam) Unwrap() any { unwrapCalls++; return w.inner }
func (WrongParam) Is(any) bool { isCalls++; return false }
type AliasError = error
type Wrapped struct { inner error }
func (Wrapped) Error() string { return "wrapped" }
func (w Wrapped) Unwrap() AliasError { unwrapCalls++; return w.inner }
func (Wrapped) Is(error) Flag { isCalls++; return false }
func optionalMethods(name string, value, target error) {
 unwrapCalls = 0; isCalls = 0
 println(name, errors.Unwrap(value) == nil, errors.Is(value, target), unwrapCalls, isCalls)
 unwrapCalls = 0; isCalls = 0
 println(name, "nil target", errors.Is(value, nil), unwrapCalls, isCalls)
}
func inspect(name string, value any) {
 e, isError := value.(error)
 marker, isRuntime := value.(RuntimeMarker)
 _, isBoth := value.(RuntimeError)
 timeout, isTimeout := value.(Timeout)
 temporary, isTemporary := value.(Temporary)
 _, isDeadline := value.(DeadlineError)
 _, isDeadlineAlias := value.(DeadlineAlias)
 _, wrongError := value.(interface { Error() int })
 _, wrongRuntime := value.(interface { RuntimeError() bool })
 _, wrongTimeout := value.(interface { Timeout() int })
 _, wrongTemporary := value.(interface { Temporary(int) bool })
 println(name, "methods", isError, isRuntime, isBoth)
 println(name, "deadline methods", isTimeout, isTemporary, isDeadline, isDeadlineAlias)
 println(name, "wrong signatures", wrongError, wrongRuntime, wrongTimeout, wrongTemporary)
 if isError {
  println(name, e.Error())
  sameText := errors.New(e.Error())
  println(name, "error type identity", e == sameText, sameText == e)
 }
 if isRuntime { marker.RuntimeError() }
 if isTimeout { bound := timeout.Timeout; println(name, "timeout", timeout.Timeout(), bound()) }
 if isTemporary { bound := temporary.Temporary; println(name, "temporary", temporary.Temporary(), bound()) }
 switch value.(type) {
 case RuntimeError: println(name, "switch runtime")
 case error: println(name, "switch error")
 default: println(name, "switch other")
 }
}
func recovered(name string, f func()) {
 defer func() { inspect(name, recover()) }()
 f()
}
func failedExtra(name string, f func()) {
 recovered(name, func() {
  defer func() { _ = recover().(Extra) }()
  f()
 })
}
func assertionError() (err error) {
 defer func() { err = recover().(error) }()
 var value any = 1
 _ = value.(string)
 return nil
}
func main() {
 ordinary := errors.New("ordinary")
 inspect("new", ordinary)
 _, capability := crypto.Random(-1)
 inspect("crypto", capability)
 inspect("canceled", context.Canceled)
 inspect("deadline", context.DeadlineExceeded)
 duplicate := errors.New("context deadline exceeded")
 println("deadline identity", context.DeadlineExceeded == context.DeadlineExceeded,
  context.DeadlineExceeded == duplicate, duplicate == context.DeadlineExceeded,
  errors.Is(context.DeadlineExceeded, duplicate), errors.Is(duplicate, context.DeadlineExceeded))
 ctx, cancel := context.WithTimeout(context.Background(), -1)
 defer cancel()
 <-ctx.Done()
 inspect("context deadline", ctx.Err())
 println("context sentinel", ctx.Err() == context.DeadlineExceeded,
  context.DeadlineExceeded == ctx.Err(), errors.Is(ctx.Err(), context.DeadlineExceeded))
 inspect("user", UserError{})
 inspect("named result", NamedError{})
 target := errors.New("target")
 optionalMethods("wrong result methods", WrongResult{}, target)
 optionalMethods("wrong parameter methods", WrongParam{target}, target)
 optionalMethods("correct methods", Wrapped{target}, target)
 var calculator any = Calculator{}
 correct, isCorrect := calculator.(interface { Calculate(int) int })
 _, wrongParam := calculator.(interface { Calculate(string) int })
 _, wrongResult := calculator.(interface { Calculate(int) string })
 _, wrongArity := calculator.(interface { Calculate() int })
 println("source signatures", isCorrect, correct.Calculate(2), wrongParam, wrongResult, wrongArity)
 first, second := assertionError(), assertionError()
 println("assertion pointer identity", first == first, first == second, second == first)
 recovered("divide", func() { zero := 0; _ = 1 / zero })
 recovered("channel", func() { ch := make(chan int); close(ch); close(ch) })
 recovered("nil panic", func() { panic(nil) })
 recovered("index", func() { values := []int{1}; index := 2; _ = values[index] })
 recovered("slice", func() { values := []int{1}; high := 2; _ = values[:high] })
 recovered("array conversion", func() { values := []int{1}; _ = [2]int(values) })
 failedExtra("nil panic extra", func() { panic(nil) })
 failedExtra("index extra", func() { values := []int{1}; index := 2; _ = values[index] })
 failedExtra("slice extra", func() { values := []int{1}; high := 2; _ = values[:high] })
 failedExtra("array conversion extra", func() { values := []int{1}; _ = [2]int(values) })
 recovered("assert primitive", func() { var value any = 1; _ = value.(string) })
 recovered("assert ordinary", func() { _ = ordinary.(RuntimeMarker) })
 recovered("assert missing", func() { _ = ordinary.(Extra) })
 recovered("assert ordinary timeout", func() { _ = ordinary.(Timeout) })
 recovered("assert ordinary signature", func() { _ = any(ordinary).(interface { Error() int }) })
 recovered("assert source signature", func() { _ = calculator.(interface { Calculate(string) int }) })
 recovered("assert deadline runtime", func() { _ = context.DeadlineExceeded.(RuntimeMarker) })
 recovered("assert runtime timeout", func() {
  defer func() { _ = recover().(Timeout) }()
  zero := 0; _ = 1 / zero
 })
 recovered("panic ordinary", func() { panic(ordinary) })
}
`,
	} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	testutil.RunFixture(t, testutil.Fixture{Name: "native_error_methods", Dir: source, Gate: "cooperative"}, []string{"swift"})
	// Go's fallback panic text can include host pointer addresses. Compare the
	// observable method calls instead: only exact Error/String signatures run.
	for _, tc := range []struct {
		name, methods string
		wantGood      bool
	}{
		{"wrong_both", `func (Value) Error() int { println("WRONG_METHOD_CALLED"); return 1 }
func (Value) String() int { println("WRONG_METHOD_CALLED"); return 2 }`, false},
		{"wrong_error_good_string", `func (Value) Error() int { println("WRONG_METHOD_CALLED"); return 1 }
func (Value) String() Message { println("GOOD_METHOD_CALLED"); return "good" }`, true},
		{"good_error_wrong_string", `func (Value) Error() Message { println("GOOD_METHOD_CALLED"); return "good" }
func (Value) String() int { println("WRONG_METHOD_CALLED"); return 2 }`, true},
	} {
		t.Run("panic_formatting_"+tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, text := range map[string]string{
				"go.mod":  fmt.Sprintf("module swiftpanicformat\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %q\n", root),
				"main.go": "package main\ntype Message = string\ntype Value struct { n int }\n" + tc.methods + "\nfunc main() { panic(Value{5}) }\n",
			} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			work := t.TempDir()
			native, err := testutil.Native(dir, work)
			if err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(work, "swift")
			if ds := testutil.CompileGateOptions(dir, "swift", out, "cooperative", driver.EmitOptions{}); len(ds) != 0 {
				t.Fatal(ds)
			}
			swift, err := testutil.Runners["swift"](out)
			if err != nil {
				t.Fatal(err)
			}
			for name, observation := range map[string]testutil.Observation{"native Go": native, "Swift": swift} {
				wantCalls := 0
				if tc.wantGood {
					wantCalls = 1
				}
				if observation.Exit != 2 || strings.Contains(observation.Stderr, "WRONG_METHOD_CALLED") || strings.Count(observation.Stderr, "GOOD_METHOD_CALLED") != wantCalls {
					t.Errorf("%s panic formatter called an incorrect method: %s", name, observation)
				}
			}
		})
	}
}
