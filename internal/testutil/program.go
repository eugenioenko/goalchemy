package testutil

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/eugenioenko/goalchemy/internal/driver"
)

// Program runs a prepared fixture in its own output directory. Only build
// artifacts are reusable; each Run starts a new process with the current env.
type Program struct {
	dir    string
	args   []string
	Reused bool
}

func (p *Program) Run(timeout time.Duration, env ...string) (Observation, error) {
	return runEnv(p.dir, timeout, env, p.args[0], p.args[1:]...)
}

// Prepare builds a freshly emitted fixture. CI provides a temporary cache
// shared by the suite's test processes, never by different jobs or commits.
func Prepare(target, out string) (*Program, error) {
	p := &Program{dir: out}
	var artifacts []string
	var build func() error
	switch target {
	case "go":
		artifacts, p.args = []string{"prog"}, []string{filepath.Join(out, "prog")}
		build = func() error {
			o, err := run(out, 2*time.Minute, "go", "build", "-o", "prog", ".")
			if err != nil || o.Exit != 0 {
				return fmt.Errorf("go build failed: %v\n%s", err, o.Stderr)
			}
			return nil
		}
	case "swift":
		artifacts, p.args = []string{".build/program"}, []string{filepath.Join(out, ".build", "program")}
		build = func() error { return BuildSwift(out) }
	case "csharp":
		artifacts = []string{"bin"}
		p.args = []string{"sh", "-c", `dotnet=dotnet; if [ -n "$DOTNET_ROOT" ] && [ -x "$DOTNET_ROOT/dotnet" ]; then dotnet="$DOTNET_ROOT/dotnet"; fi; exec "$dotnet" bin/main.dll`}
		build = func() error { return BuildCSharp(out) }
	case "java":
		artifacts = []string{"classes"}
		var err error
		p.args, err = javaCommand(out)
		if err != nil {
			return nil, err
		}
		p.args = append([]string{"sh", "-c", `if [ -n "$JAVA_HOME" ]; then PATH="$JAVA_HOME/bin:$PATH"; fi; exec "$@"`, "goalchemy-java"}, p.args...)
		build = func() error { _, err := BuildJava(out); return err }
	case "rust":
		artifacts, p.args = []string{"main"}, []string{filepath.Join(out, "main")}
		build = func() error { _, err := BuildRust(out); return err }
	case "typescript":
		p.args = []string{"node", "--stack-size=4000", "main.ts"}
		return p, nil
	case "python":
		p.args = []string{"python3", "main.py"}
		return p, nil
	default:
		return nil, fmt.Errorf("no fixture preparation for %s", target)
	}
	var err error
	p.Reused, err = cachedBuild(target, out, artifacts, build)
	return p, err
}

var artifactLocks sync.Map

func cachedBuild(target, out string, artifacts []string, build func() error) (bool, error) {
	root := os.Getenv("GOALCHEMY_TEST_BUILD_CACHE")
	if root == "" {
		return false, build()
	}
	key, err := buildKey(target, out, driver.ToolEnv())
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return false, err
	}
	entry := filepath.Join(root, key)
	mu, _ := artifactLocks.LoadOrStore(entry, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()
	unlock, err := lockFile(entry + ".lock")
	if err != nil {
		return false, err
	}
	defer unlock()
	if _, err := os.Stat(entry); err == nil {
		return true, copyArtifacts(entry, out, artifacts)
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := build(); err != nil {
		return false, err
	}
	stage, err := os.MkdirTemp(root, ".build-")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(stage)
	if err := copyArtifacts(out, stage, artifacts); err != nil {
		return false, err
	}
	return false, os.Rename(stage, entry)
}

func buildKey(target, out string, env []string) (string, error) {
	h := sha256.New()
	fmt.Fprintf(h, "fixture-build-v1\n%s\n%s\n%s\n", target, runtime.GOOS, runtime.GOARCH)
	filtered := make([]string, 0, len(env))
	for _, value := range env {
		key, _, _ := strings.Cut(value, "=")
		switch key {
		case "GOALCHEMY_GC_THRESHOLD", "GOALCHEMY_SEED", "FIXTURE":
			continue
		}
		filtered = append(filtered, value)
	}
	sort.Strings(filtered)
	for _, value := range filtered {
		fmt.Fprintf(h, "%q\n", value)
	}
	err := filepath.WalkDir(out, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("fixture build input is not a regular file: %s", path)
		}
		rel, err := filepath.Rel(out, path)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%q %d %d\n", filepath.ToSlash(rel), info.Mode().Perm(), info.Size())
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(h, file)
		closeErr := file.Close()
		if err != nil {
			return err
		}
		return closeErr
	})
	return fmt.Sprintf("%x", h.Sum(nil)), err
}

func copyArtifacts(from, to string, artifacts []string) error {
	for _, name := range artifacts {
		src, dst := filepath.Join(from, name), filepath.Join(to, name)
		info, err := os.Stat(src)
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, info.Mode().Perm()); err != nil {
			return err
		}
	}
	return nil
}
