package contracts

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eugenioenko/goalchemy/internal/driver"
)

// Go source cannot construct distinct runtime wrappers sharing a Swift Array.
// Check that COW snapshots preserve both those wrappers and Go backing aliases.
func TestSwiftBulkByteRuntime(t *testing.T) {
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
 GType("uint8","int",8,false,-1,-1,0,[],[],[],[],true),
 GType("Octet","int",8,false,-1,-1,0,[],[],[],[],true),
 GType("int","int",64,true,-1,-1,0,[],[],[],[],true)
]
func slice(_ b: GBuffer?, _ lo: Int, _ n: Int, _ cap: Int, _ elem: Int = 0) -> GValue {
 .slice(GSlice(b,lo,n,cap,elem))
}
let original: [UInt8] = [1,2,3,4,5,6]
let a=GBuffer(bytes:original,elem:0),b=GBuffer(bytes:original,elem:0)
require(try GSlice.copy(slice(a,1,4,5),slice(b,0,4,6)).intValue==4,"copy count")
require(a.bytes==[1,1,2,3,4,6] && b.bytes==original && original==[1,2,3,4,5,6],"distinct COW wrappers")
let alias=slice(a,0,6,6)
_ = try GSlice.copy(slice(a,0,4,6),slice(a,2,4,4))
require(try GNative.buffer(alias)==[2,3,4,6,4,6],"backward overlap and shared alias")
let appended=try GNative.invoke("core.slice.append",[slice(a,1,1,5),slice(a,0,4,6)])[0]
require(try GNative.buffer(appended)==[3,2,3,4,6],"append source snapshot")
require(a.bytes==[2,3,2,3,4,6],"append in-place alias")
let grown=try GSlice.append(slice(a,2,2,2),slice(a,0,1,6))
guard case .slice(let g)=grown else { fatalError("grown slice") }
require(g.offset==0 && g.length==3 && g.capacity==4 && g.storage !== a,"growth header")
require(g.storage!.bytes==[2,3,2,0] && a.bytes==[2,3,2,3,4,6],"growth prefix and zero spare capacity")
_ = try GSlice.append(slice(a,1,1,5),[.integer(255,8,false),.integer(0,8,false)])
require(a.bytes==[2,3,255,0,4,6],"scalar append on offset")
let named=GBuffer(3,1)
require(try GSlice.copy(slice(named,0,2,3,1),.string([255,0,128])).intValue==2,"named byte short string copy")
require(named.bytes==[255,0,0],"named byte output")
let nilBytes=slice(nil,0,0,0)
guard case .slice(let nilResult)=try GSlice.append(nilBytes,nilBytes) else{fatalError("nil append")}
require(nilResult.storage==nil,"nil append changed identity")
let empty=GBuffer(0,0)
guard case .slice(let emptyResult)=try GSlice.append(slice(empty,0,0,0),nilBytes) else{fatalError("empty append")}
require(emptyResult.storage===empty,"empty append discarded storage")
require(try GSlice.copy(nilBytes,.string([1])).intValue==0,"nil copy")
let values=GBuffer(3,2);values.write(0,.int(7));values.write(1,.int(8));values.write(2,.int(9))
require(try GSlice.copy(slice(values,1,2,2,2),slice(values,0,2,3,2)).intValue==2,"generic copy")
require(values.read(2).intValue==8,"generic overlap")
do { _ = try GSlice.append(.int(1),.int(2));fatalError("bad append accepted") }
catch let e as GFault {require(e.message=="expected slice, array or string","append validation order")}
do { _ = try GSlice.copy(.int(1),.int(2));fatalError("bad copy accepted") }
catch let e as GFault {require(e.message=="slice representation","copy validation order")}
let large=GBuffer(bytes:[UInt8](repeating:128,count:2 << 20),elem:0)
let big=try GSlice.append(slice(nil,0,0,0),slice(large,1,(2 << 20)-1,(2 << 20)-1))
require(try GNative.buffer(big).count==(2 << 20)-1,"large packed append")
GHeap.collect([big,alias])
require(try GNative.buffer(big).last==128 && GNative.buffer(alias)[2]==255,"collection lost byte storage")
print("PASS Swift bulk bytes")
`
	if err := os.WriteFile(filepath.Join(dir, "main.swift"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", `cc -Wno-deprecated-declarations -O2 -c rt/native/GoalchemyNative.c -o native.o $(pkg-config --cflags openssl zlib libcurl) && swiftc -swift-version 5 -suppress-warnings -I rt/native main.swift rt/types/*.swift rt/runtime/*.swift native.o $(pkg-config --libs openssl zlib libcurl) -o probe && ./probe`)
	cmd.Dir, cmd.Env = dir, driver.ToolEnv()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native bulk byte regression: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "PASS Swift bulk bytes") {
		t.Fatal(string(output))
	}
}
