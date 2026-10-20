package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eugenioenko/goalchemy/internal/link"
)

// The common types contain no generated declarations. Compiling their object
// once keeps every fixture in the same module without importing a test runtime.
func buildSwiftObjects(out string) error {
	data, err := os.ReadFile(filepath.Join(out, "goalchemy.manifest.json"))
	if err != nil {
		return err
	}
	var manifest link.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}
	var types, sources []string
	for _, name := range append(manifest.GeneratedFiles, manifest.RuntimeFiles...) {
		if !strings.HasSuffix(name, ".swift") {
			continue
		}
		if strings.HasPrefix(name, "rt/types/") {
			types = append(types, name)
		} else {
			sources = append(sources, name)
		}
	}
	if len(types) == 0 || len(sources) == 0 {
		return fmt.Errorf("Swift manifest has no types or program sources")
	}
	support := filepath.Join(out, ".build", "support")
	if err := os.RemoveAll(support); err != nil {
		return err
	}
	if err := copyArtifacts(out, support, []string{"rt/types", "rt/native"}); err != nil {
		return err
	}
	flags := []string{"-swift-version", "5", "-suppress-warnings", "-module-name", "program", "-I", "rt/native"}
	_, err = cachedBuild("swift-support-v1", support, []string{"types.o", "native.o"}, func() error {
		if err := swiftBuildCommand(support, "sh", "-c", `exec "${CC:-cc}" -O2 -Wno-deprecated-declarations -c rt/native/GoalchemyNative.c -o native.o $(pkg-config --cflags openssl zlib libcurl)`); err != nil {
			return err
		}
		args := append([]string{}, flags...)
		args = append(args, "-parse-as-library", "-whole-module-optimization", "-emit-object")
		args = append(args, types...)
		return swiftc(support, append(args, "-o", "types.o")...)
	})
	if err != nil {
		return err
	}
	args := append([]string{"-frontend", "-c"}, flags...)
	var objects []string
	for i, name := range sources {
		args = append(args, "-primary-file", name)
		objects = append(objects, filepath.Join(".build", fmt.Sprintf("source-%d.o", i)))
	}
	args = append(args, types...)
	for _, object := range objects {
		args = append(args, "-o", object)
	}
	if err := swiftc(out, args...); err != nil {
		return err
	}
	objects = append(objects, ".build/support/types.o", ".build/support/native.o")
	return swiftBuildCommand(out, "sh", append([]string{"-c", `exec "${SWIFTC:-swiftc}" "$@" $(pkg-config --libs openssl zlib libcurl) -o .build/program`, "goalchemy-swift"}, objects...)...)
}

func swiftc(out string, args ...string) error {
	return swiftBuildCommand(out, "sh", append([]string{"-c", `exec "${SWIFTC:-swiftc}" "$@"`, "goalchemy-swift"}, args...)...)
}

func swiftBuildCommand(out, name string, args ...string) error {
	o, err := run(out, 5*time.Minute, name, args...)
	if err != nil || o.Exit != 0 {
		return fmt.Errorf("Swift build: %v\n%s", err, o.Stderr)
	}
	return nil
}
