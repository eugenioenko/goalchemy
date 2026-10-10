package contracts

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/testutil"
)

const filesProbe = `package main

import "github.com/eugenioenko/goalchemy/std/os"

func check(err error) {
	if err != nil {
		panic(err.Error())
	}
}

func main() {
	check(os.WriteFile("fresh.txt", []byte("fresh"), 0o640))
	check(os.WriteFile("kept.txt", []byte("first"), 0o600))
	check(os.WriteFile("kept.txt", []byte("second"), 0o644))
	check(os.WriteFile("wide.txt", []byte("wide"), 0o777))
	check(os.WriteFile("high.txt", []byte("high"), 0o4755))
}
`

// Every target applies WriteFile permissions to new files only, after the
// process umask, and keeps the mode of files it truncates.
func TestFilePermissions(t *testing.T) {
	umask := os.FileMode(syscall.Umask(0))
	syscall.Umask(int(umask))
	root, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	files := map[string]string{
		"main.go": filesProbe,
		"go.mod":  fmt.Sprintf("module filesprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", strconv.Quote(root)),
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(source, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	targets := []string{"go", "typescript", "python", "java", "csharp", "rust", "c", "swift"}
	if only := os.Getenv("GOALCHEMY_TEST_TARGETS"); only != "" {
		targets = strings.Split(only, ",")
	}
	want := map[string]struct {
		mode os.FileMode
		data string
	}{
		"fresh.txt": {0o640 &^ umask, "fresh"},
		"kept.txt":  {0o600 &^ umask, "second"},
		"wide.txt":  {0o777 &^ umask, "wide"},
		"high.txt":  {0o755 &^ umask, "high"},
	}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			t.Parallel()
			out := t.TempDir()
			if ds := testutil.Compile(source, target, out); len(ds) != 0 {
				t.Fatal(ds)
			}
			observation, err := testutil.Runners[target](out)
			if err != nil || observation.Exit != 0 {
				t.Fatalf("%s probe: %v\n%s", target, err, observation)
			}
			for name, w := range want {
				path := filepath.Join(out, name)
				st, err := os.Stat(path)
				if err != nil {
					t.Fatalf("%s: %v", name, err)
				}
				data, _ := os.ReadFile(path)
				if st.Mode() != w.mode || string(data) != w.data {
					t.Errorf("%s: mode %v data %q; want %v %q", name, st.Mode(), data, w.mode, w.data)
				}
			}
		})
	}
}
