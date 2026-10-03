package integration

import (
	"bytes"
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

func TestGeneratedCSharpChecksumEntries(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	write := func(t *testing.T, dir, name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(t *testing.T, dir, name string, args ...string) (string, string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir, cmd.Env = dir, driver.ToolEnv()
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("%s: %v\n%s\n%s", name, err, stdout.String(), stderr.String())
		}
		return stdout.String(), stderr.String()
	}
	mod := fmt.Sprintf("module crcprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root)
	t.Run("executable", func(t *testing.T) {
		source, out := t.TempDir(), t.TempDir()
		write(t, source, "go.mod", mod)
		write(t, source, "main.go", `package main
import "github.com/eugenioenko/goalchemy/lib/checksum"
func main(){
 var missing []byte
 println(checksum.CRC32IEEE(missing),checksum.CRC32IEEE([]byte{}))
 bytes:=[]byte("x123456789y")
 println(checksum.CRC32IEEE(bytes[1:10]))
 if string(bytes)!="x123456789y"{panic("input mutated")}
}`)
		if ds := testutil.Compile(source, "csharp", out); len(ds) > 0 {
			t.Fatal(ds)
		}
		stdout, stderr := run(t, out, "sh", "run.sh")
		if stdout != "" || stderr != "0 0\n3421780262\n" {
			t.Fatalf("exact script output: stdout=%q stderr=%q", stdout, stderr)
		}
		// Also exercise normal SDK dependency resolution, independently of run.sh.
		run(t, out, "dotnet", "build", "main.csproj", "--nologo")
		stdout, stderr = run(t, out, "dotnet", "run", "--project", "main.csproj", "--no-build")
		if stdout != "" || stderr != "0 0\n3421780262\n" {
			t.Fatalf("exact SDK output: stdout=%q stderr=%q", stdout, stderr)
		}
	})
	t.Run("library-project-consumer", func(t *testing.T) {
		source, out, consumer := t.TempDir(), t.TempDir(), t.TempDir()
		write(t, source, "go.mod", mod)
		write(t, source, "library.go", `package crcprobe
import("github.com/eugenioenko/goalchemy/lib/checksum";"github.com/eugenioenko/goalchemy/lib/context")
func CRC32IEEE(ctx context.Context,data []byte)(uint32,error){return checksum.CRC32IEEE(data),nil}
func SliceCRC32IEEE(ctx context.Context,data []byte)(uint32,error){return checksum.CRC32IEEE(data[1:10]),nil}
`)
		if ds := testutil.CompileGate(source, "csharp", out, "cooperative"); len(ds) > 0 {
			t.Fatal(ds)
		}
		run(t, out, "sh", "build.sh")
		write(t, consumer, "consumer.csproj", `<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><OutputType>Exe</OutputType><TargetFramework>net8.0</TargetFramework></PropertyGroup><ItemGroup><ProjectReference Include="`+filepath.Join(out, "main.csproj")+`" /></ItemGroup></Project>`)
		write(t, consumer, "Consumer.cs", `using System;
using System.IO;
using System.Linq;
using System.Reflection;
using Rt;
using Generated=Goalchemy.Generated.GoProgram;
class Consumer {
 static long CRC(byte[] bytes)=>Generated.CRC32IEEE(bytes,null).Completion.GetAwaiter().GetResult();
 static void Main(){
  if(CRC(null)!=0||CRC(Array.Empty<byte>())!=0)throw new Exception("nil/empty");
  var bytes=new byte[]{120,49,50,51,52,53,54,55,56,57,121};var before=(byte[])bytes.Clone();
  if(Generated.SliceCRC32IEEE(bytes,null).Completion.GetAwaiter().GetResult()!=0xcbf43926L||!bytes.SequenceEqual(before))throw new Exception("slice/unsigned/ownership");
  // A normal project reference must carry the official package transitively.
  var native=typeof(System.IO.Hashing.Crc32).Assembly;
  if(native.GetName().Version.Major!=8||Path.GetDirectoryName(native.Location)!=AppContext.BaseDirectory.TrimEnd(Path.DirectorySeparatorChar))throw new Exception("normal package deployment");
  // Confirm compiler output calls the public native API, not a private fallback.
  var method=typeof(R).GetMethod("libChecksumCRC32IEEE");var il=method.GetMethodBody().GetILAsByteArray();bool found=false;
  for(int i=0;i+4<il.Length;i++)if(il[i]==0x28){try{var called=method.Module.ResolveMethod(BitConverter.ToInt32(il,i+1));found|=called.DeclaringType==typeof(System.IO.Hashing.Crc32)&&called.Name=="HashToUInt32";}catch(ArgumentException){}}
  if(!found)throw new Exception("native API call missing");
  Console.WriteLine("PASS generated C# library native CRC and transitive package deployment");
 }
}`)
		run(t, consumer, "dotnet", "build", "consumer.csproj", "--nologo", "-o", "bin")
		stdout, _ := run(t, consumer, "dotnet", "bin/consumer.dll")
		if !strings.Contains(stdout, "PASS generated C# library native CRC") {
			t.Fatal(stdout)
		}
		t.Log(stdout)
	})
	t.Run("without-checksum", func(t *testing.T) {
		source, out := t.TempDir(), t.TempDir()
		write(t, source, "go.mod", mod)
		write(t, source, "main.go", "package main\nfunc main(){println(7)}\n")
		if ds := testutil.Compile(source, "csharp", out); len(ds) > 0 {
			t.Fatal(ds)
		}
		for _, file := range []string{"run.sh", "main.csproj"} {
			data, err := os.ReadFile(filepath.Join(out, file))
			if err != nil || strings.Contains(string(data), "System.IO.Hashing") || strings.Contains(string(data), "restore") {
				t.Fatalf("unexpected dependency in %s: %v", file, err)
			}
		}
		if _, err := os.Stat(filepath.Join(out, "packages.lock.json")); !os.IsNotExist(err) {
			t.Fatal("unused checksum dependency lock", err)
		}
		stdout, stderr := run(t, out, "sh", "run.sh")
		if stdout != "" || stderr != "7\n" {
			t.Fatalf("exact dependency-free script output: stdout=%q stderr=%q", stdout, stderr)
		}
	})
}
