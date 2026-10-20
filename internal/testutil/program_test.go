package testutil

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func buildInput(t *testing.T) string {
	t.Helper()
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "source"), []byte("same source"), 0o644); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestBuildKey(t *testing.T) {
	first, second := buildInput(t), buildInput(t)
	key := func(target, out string, env ...string) string {
		t.Helper()
		key, err := buildKey(target, out, env)
		if err != nil {
			t.Fatal(err)
		}
		return key
	}
	base := key("swift", first, "SWIFTC=swiftc", "CC=cc")
	if got := key("swift", second, "CC=cc", "SWIFTC=swiftc", "GOALCHEMY_GC_THRESHOLD=1", "GOALCHEMY_SEED=7", "FIXTURE=co_"); got != base {
		t.Fatal("runtime settings, env order or output directory changed the build key")
	}
	if key("rust", second, "CC=cc", "SWIFTC=swiftc") == base || key("swift", second, "CC=clang", "SWIFTC=swiftc") == base {
		t.Fatal("target or build environment did not change the key")
	}
	for _, name := range []string{"source", "runtime", "run.sh", "goalchemy.manifest.json"} {
		if err := os.WriteFile(filepath.Join(second, name), []byte("changed"), 0o644); err != nil {
			t.Fatal(err)
		}
		if key("swift", second, "CC=cc", "SWIFTC=swiftc") == base {
			t.Fatalf("changed %s reused a key", name)
		}
		if name == "source" {
			os.WriteFile(filepath.Join(second, name), []byte("same source"), 0o644)
		} else {
			os.Remove(filepath.Join(second, name))
		}
	}
}

func TestCachedBuildRunsFreshAndKeepsDirectoriesPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture shell")
	}
	t.Setenv("GOALCHEMY_TEST_BUILD_CACHE", t.TempDir())
	var builds int
	prepare := func(out string) *Program {
		t.Helper()
		reused, err := cachedBuild("test", out, []string{"bin"}, func() error {
			builds++
			if err := os.Mkdir(filepath.Join(out, "bin"), 0o755); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(out, "bin", "program"), []byte("#!/bin/sh\nprintf '%s %s\\n' \"$GOALCHEMY_GC_THRESHOLD\" \"$GOALCHEMY_SEED\"\nif [ -f touched ]; then exit 9; fi\ntouch touched\n"), 0o755)
		})
		if err != nil {
			t.Fatal(err)
		}
		return &Program{dir: out, args: []string{filepath.Join(out, "bin", "program")}, Reused: reused}
	}
	first := prepare(buildInput(t))
	obs, err := first.Run(time.Second, "GOALCHEMY_GC_THRESHOLD=0", "GOALCHEMY_SEED=1")
	if err != nil || obs.Exit != 0 || obs.Stdout != "0 1\n" || first.Reused {
		t.Fatalf("first run: %v %v", obs, err)
	}
	t.Setenv("GOALCHEMY_GC_THRESHOLD", "1")
	t.Setenv("GOALCHEMY_SEED", "4242")
	second := prepare(buildInput(t))
	obs, err = second.Run(time.Second)
	if err != nil || obs.Exit != 0 || obs.Stdout != "1 4242\n" || !second.Reused || builds != 1 {
		t.Fatalf("reused run: %v %v, builds=%d reused=%v", obs, err, builds, second.Reused)
	}
	obs, err = second.Run(time.Second, "GOALCHEMY_SEED=7")
	if err != nil || obs.Exit != 9 || obs.Stdout != "1 7\n" {
		t.Fatalf("run output or failure was cached: %v %v", obs, err)
	}
}

func TestCachedBuildConcurrentAndFailure(t *testing.T) {
	t.Setenv("GOALCHEMY_TEST_BUILD_CACHE", t.TempDir())
	var builds atomic.Int32
	inputs := make([]string, 8)
	for i := range inputs {
		inputs[i] = buildInput(t)
	}
	var wg sync.WaitGroup
	for _, out := range inputs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cachedBuild("test", out, []string{"program"}, func() error {
				builds.Add(1)
				return os.WriteFile(filepath.Join(out, "program"), []byte("binary"), 0o755)
			})
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if builds.Load() != 1 {
		t.Fatalf("built %d times", builds.Load())
	}
	failure := errors.New("build failed")
	if reused, err := cachedBuild("failure", buildInput(t), []string{"program"}, func() error { return failure }); reused || !errors.Is(err, failure) {
		t.Fatalf("failed build: reused=%v err=%v", reused, err)
	}
	if reused, err := cachedBuild("failure", buildInput(t), []string{"program"}, func() error { return failure }); reused || !errors.Is(err, failure) {
		t.Fatalf("failure was cached: reused=%v err=%v", reused, err)
	}
}

func TestBuildWithoutCache(t *testing.T) {
	t.Setenv("GOALCHEMY_TEST_BUILD_CACHE", "")
	builds := 0
	for range 2 {
		reused, err := cachedBuild("test", "", nil, func() error { builds++; return nil })
		if err != nil || reused {
			t.Fatalf("uncached build: reused=%v err=%v", reused, err)
		}
	}
	if builds != 2 {
		t.Fatalf("uncached build executed %d times", builds)
	}
}

func TestCachedBuildAcrossProcesses(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file locks are Unix only")
	}
	root := t.TempDir()
	t.Setenv("GOALCHEMY_TEST_BUILD_CACHE", root)
	commands := make([]*exec.Cmd, 4)
	for i := range commands {
		cmd := exec.Command(os.Args[0], "-test.run=^TestCachedBuildHelperProcess$", "--", buildInput(t))
		cmd.Env = append(os.Environ(), "GOALCHEMY_BUILD_HELPER=1")
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		commands[i] = cmd
	}
	for _, cmd := range commands {
		if err := cmd.Wait(); err != nil {
			t.Error(err)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, "count"))
	if err != nil || string(data) != "build\n" {
		t.Fatalf("cross-process builds: %q, %v", data, err)
	}
}

func TestCachedBuildHelperProcess(t *testing.T) {
	if os.Getenv("GOALCHEMY_BUILD_HELPER") == "" {
		return
	}
	out := os.Args[len(os.Args)-1]
	_, err := cachedBuild("test", out, []string{"program"}, func() error {
		count, err := os.OpenFile(filepath.Join(os.Getenv("GOALCHEMY_TEST_BUILD_CACHE"), "count"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return err
		}
		_, err = count.WriteString("build\n")
		count.Close()
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(out, "program"), []byte("binary"), 0o755)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestProgramTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture shell")
	}
	p := &Program{dir: t.TempDir(), args: []string{"sh", "-c", "exec sleep 5"}}
	_, err := p.Run(20 * time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("timeout: %v", err)
	}
}

func TestPreparedRuntimeUsesToolchainEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture shell")
	}
	t.Chdir(t.TempDir())
	t.Setenv("GOALCHEMY_TEST_BUILD_CACHE", "")
	tools := t.TempDir()
	if err := os.Mkdir(filepath.Join(tools, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"bin/java", "dotnet"} {
		if err := os.WriteFile(filepath.Join(tools, name), []byte("#!/bin/sh\nprintf '%s\\n' \"$*\"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("JAVA_HOME", tools)
	t.Setenv("DOTNET_ROOT", tools)
	for _, tc := range []struct{ target, script, want string }{
		{"java", "mkdir -p classes\nexec java -Xss1g -cp classes Main\n", "-Xss1g -cp classes Main\n"},
		{"csharp", "mkdir -p bin\nexec dotnet bin/main.dll\n", "bin/main.dll\n"},
	} {
		t.Run(tc.target, func(t *testing.T) {
			out := t.TempDir()
			if err := os.WriteFile(filepath.Join(out, "run.sh"), []byte(tc.script), 0o644); err != nil {
				t.Fatal(err)
			}
			program, err := Prepare(tc.target, out)
			if err != nil {
				t.Fatal(err)
			}
			got, err := program.Run(time.Second)
			if err != nil || got.Exit != 0 || got.Stdout != tc.want {
				t.Fatalf("toolchain runtime: %v %v", got, err)
			}
		})
	}
}
