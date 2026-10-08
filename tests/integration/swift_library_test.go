package integration

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

// Import the public SwiftPM product, independently of compiler-private names.
// This exercises submission copies, foreign-thread completion, cancellation
// acknowledgement, callback reentry rejection and retained native key aliases.
func TestSwiftLibraryHostLifecycle(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source, out, consumer := t.TempDir(), t.TempDir(), t.TempDir()
	write := func(dir, name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(source, "go.mod", fmt.Sprintf("module swifthostprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %q\n", root))
	write(source, "library.go", `package hostprobe
import (
 "github.com/eugenioenko/goalchemy/lib/context"
 "github.com/eugenioenko/goalchemy/lib/errors"
 "github.com/eugenioenko/goalchemy/lib/callback"
 "github.com/eugenioenko/goalchemy/lib/crypto"
 "github.com/eugenioenko/goalchemy/lib/task"
 "github.com/eugenioenko/goalchemy/lib/time"
)
func Request(ctx context.Context, bytes []byte)([]byte,error){return callback.Request(ctx,"echo",bytes)}
func OtherWork(ctx context.Context)([]byte,error){
 child,cancel:=context.WithCancel(ctx)
 var result []byte
 task.All(func(){callback.Request(child,"block",nil)},func(){time.Sleep(time.Millisecond);cancel();result,_=callback.Request(ctx,"probe",nil)})
 return result,nil
}
func Identity(value string)(string,error){return value,nil}
func ErrorBytes(value string)(string,error){return errors.New(value).Error(),nil}
func Key()(*crypto.Key,error){return crypto.GenerateP256()}
func Public(key *crypto.Key)(string,error){return key.PublicPEM()}
func Same(a,b *crypto.Key)(bool,error){return a==b,nil}
func Alias(key *crypto.Key)(*crypto.Key,error){return key,nil}
func Close(key *crypto.Key)error{key.Close();return nil}
func Panic()error{panic("deliberate")}
`)
	if ds := testutil.CompileGate(source, "swift", out, "cooperative"); len(ds) > 0 {
		t.Fatal(ds)
	}
	write(consumer, "Package.swift", fmt.Sprintf(`// swift-tools-version: 6.0
import PackageDescription
let package = Package(name: "HostConsumer", dependencies: [.package(name: "GoalchemyGenerated",path: %q)],targets:[.executableTarget(name:"HostConsumer",dependencies:[.product(name:"GoalchemyGenerated",package:"GoalchemyGenerated")],path:".",sources:["main.swift"])],swiftLanguageModes:[.v5])
`, out))
	write(consumer, "main.swift", `import Foundation
import GoalchemyGenerated
func require(_ value: Bool,_ text: String) { if !value {fatalError(text)} }
struct ProviderError: Error {}
let arbitrary=GoString(bytes:[255,0,128])
require(try Identity(arbitrary).wait()==arbitrary,"lossless GoString boundary")
require(try ErrorBytes(arbitrary).wait()==arbitrary,"lossless errors.New string")
var request:[UInt8]?=[255,0,128]
let echo:CallbackProvider={input,settle in
 require(input.bytes==[255,0,128],"submission input was not copied")
 DispatchQueue.global().async{settle(.success(input.bytes));settle(.success([99]))}
 return nil
}
let operation=Request(request,CallOptions(callbacks:["echo":echo]))
request![0]=99
require(try operation.wait()==[255,0,128],"foreign completion failed")
require(try operation.wait()==[255,0,128],"multiple waits changed result")
let started=DispatchSemaphore(value:0),lock=NSLock()
var cleanupCount=0
let delayed:CallbackProvider={input,settle in
 started.signal()
 return {
  do{_ = try Identity("cancel hook").wait();fatalError("cancel hook reentry permitted")}catch let error as GoalchemyFailure{require(error.kind=="host","cancel hook reentry classification")}catch{fatalError("unexpected cancel hook reentry error")}
  DispatchQueue.global().asyncAfter(deadline:.now()+0.01){
   require(input.cancellation.isCanceled,"provider cancellation token")
   lock.lock();cleanupCount+=1;lock.unlock();settle(.failure(ProviderError()))
  }
 }
}
let canceled=Request([1],CallOptions(callbacks:["echo":delayed]))
DispatchQueue.global().async{started.wait();canceled.cancel()}
do{_ = try canceled.wait();fatalError("canceled request succeeded")}catch let error as GoalchemyFailure{require(error.message=="context canceled","cancellation identity")}
lock.lock();require(cleanupCount==1,"request returned before native cleanup");lock.unlock()
let timed=Request([1],CallOptions(timeoutNanoseconds:2_000_000,callbacks:["echo":delayed]))
do{_ = try timed.wait();fatalError("deadline request succeeded")}catch let error as GoalchemyFailure{require(error.message=="context deadline exceeded","deadline identity")}
lock.lock();require(cleanupCount==2,"deadline returned before native cleanup");lock.unlock()
let reenter:CallbackProvider={input,settle in
 do{_ = try Identity("reenter").wait();fatalError("reentrant provider call permitted")}catch let error as GoalchemyFailure{require(error.kind=="host","reentry must be a host fault")}catch{fatalError("unexpected reentry error") }
 settle(.success(input.bytes));return nil
}
require(try Request([8],CallOptions(callbacks:["echo":reenter])).wait()==[8],"reentry guard broke callback completion")
let progressed=DispatchSemaphore(value:0)
let blockingHook:CallbackProvider={_,settle in
 return { require(progressed.wait(timeout:.now()+2) == .success,"blocking cleanup hook stopped source scheduler");settle(.success([])) }
}
let probe:CallbackProvider={_,settle in progressed.signal();settle(.success([9]));return nil }
require(try OtherWork(CallOptions(callbacks:["block":blockingHook,"probe":probe])).wait()==[9],"source progress during host cleanup")
let key=try Key().wait();require(key != nil,"key generation")
let pem=try Public(key).wait(),alias=try Alias(key).wait()
require(try Same(key,key).wait(),"same public key lost pointer identity")
require(try Same(nil,nil).wait() && !Same(key,nil).wait(),"nil public key pointer comparisons")
require(try Same(key,alias).wait(),"public key alias lost source pointer identity")
let distinct=try Key().wait();require(try !Same(key,distinct).wait(),"different source keys compared equal")
for _ in 0..<80 {_ = try Identity("heap boundary").wait()}
require(try Same(key,key).wait() && Same(key,alias).wait(),"public key identity lost after heap collection")
require(try Public(alias).wait()==pem,"retained native key cleared at later boundary")
try Close(key).wait()
do{_ = try Public(alias).wait();fatalError("closed alias remained usable")}catch let error as GoalchemyFailure{require(error.message=="crypto: key is closed","key aliases share close state")}
require(try Same(key,alias).wait(),"closing key changed pointer identity")
try Close(alias).wait();try Close(distinct).wait();key?.close()
do{try Panic().wait();fatalError("source panic lost")}catch let error as GoalchemyFailure{require(error.kind=="panic","source panic classified incorrectly")}
require(try Identity("after panic").wait()==GoString("after panic"),"source owner not reusable after panic")
print("PASS Swift library lifecycle")
`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", "swift run -c release")
	cmd.Dir = consumer
	cmd.Env = driver.ToolEnv()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("importing Swift consumer: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "PASS Swift library lifecycle") {
		t.Fatal(string(output))
	}
}
