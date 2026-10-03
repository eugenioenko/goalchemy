package contracts

import (
	"context"
	"github.com/eugenioenko/goalchemy/internal/driver"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCSharpNativeCrypto(t *testing.T) {
	root, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	env := driver.ToolEnv()
	dotnet := filepath.Join(root, ".toolchains/dotnet/dotnet")
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	var paths []string
	for _, dir := range []string{"types", "runtime"} {
		files, e := filepath.Glob(filepath.Join(root, "targets/csharp", dir, "*.cs"))
		if e != nil {
			t.Fatal(e)
		}
		paths = append(paths, files...)
	}
	paths = append(paths, filepath.Join(root, "targets/csharp/tests/CryptoTest.cs"))
	var project strings.Builder
	project.WriteString(`<Project Sdk="Microsoft.NET.Sdk"><PropertyGroup><OutputType>Exe</OutputType><TargetFramework>net8.0</TargetFramework><EnableDefaultCompileItems>false</EnableDefaultCompileItems><NoWarn>CS8981</NoWarn></PropertyGroup><ItemGroup>`)
	for _, p := range paths {
		project.WriteString(`<Compile Include="` + p + `" />`)
	}
	project.WriteString(`</ItemGroup><PropertyGroup><RestoreLockedMode>true</RestoreLockedMode></PropertyGroup><ItemGroup><PackageReference Include="System.IO.Hashing" Version="[8.0.0]" /></ItemGroup></Project>`)
	lock, e := os.ReadFile(filepath.Join(root, "targets/csharp/packages.lock.json"))
	if e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(out, "packages.lock.json"), lock, 0600); e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(out, "test.csproj")
	if e := os.WriteFile(file, []byte(project.String()), 0600); e != nil {
		t.Fatal(e)
	}
	cmd := exec.CommandContext(ctx, dotnet, "build", file, "-o", filepath.Join(out, "bin"), "--nologo")
	cmd.Env = env
	if data, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("native .NET compile: %v\n%s", e, data)
	}
	fixture, verify := javaCryptoFixtures(t) // independent Go vectors, shared cross-library verification
	cmd = exec.CommandContext(ctx, dotnet, filepath.Join(out, "bin/test.dll"), fixture)
	cmd.Env = env
	if data, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("native .NET vectors: %v\n%s", e, data)
	} else {
		t.Log(string(data))
	}
	verify()
}
