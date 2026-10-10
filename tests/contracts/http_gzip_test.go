package contracts

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/eugenioenko/goalchemy/internal/testutil"
)

func gzipped(parts ...string) []byte {
	var out bytes.Buffer
	for _, p := range parts {
		z := gzip.NewWriter(&out)
		z.Write([]byte(p))
		z.Close()
	}
	return out.Bytes()
}

// TestHTTPGzipDecoding compares automatic gzip decoding of lib/http with Go's
// transport on every target, against a local server.
func TestHTTPGzipDecoding(t *testing.T) {
	single := gzipped("goalchemy gzip body")
	multi := gzipped("first member;", "second member")
	large := gzipped(strings.Repeat("z", 4096))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/single", "/explicit":
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(single)
		case "/multi":
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(multi)
		case "/large":
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(large)
		case "/truncated":
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(single[:len(single)-6])
		case "/empty":
			w.Header().Set("Content-Encoding", "gzip")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	program := fmt.Sprintf(`package main

import (
	"github.com/eugenioenko/goalchemy/lib/context"
	"github.com/eugenioenko/goalchemy/lib/http"
	"github.com/eugenioenko/goalchemy/std/encoding/hex"
)

const base = %s

func show(name string, accept bool) {
	var h []string
	if accept {
		h = []string{"Accept-Encoding", "gzip"}
	}
	status, got, body, err := http.Do(context.Background(), "GET", base+name, h, nil, 1024, 5000)
	if err != nil {
		println(name, "error", status, got == nil, body == nil)
		return
	}
	encoding := ""
	for i := 0; i < len(got); i += 2 {
		if got[i] == "Content-Encoding" {
			encoding = got[i+1]
		}
	}
	println(name, status, len(body), hex.EncodeToString(body), "encoding="+encoding)
}

func main() {
	show("/single", false)
	show("/multi", false)
	show("/large", false)
	show("/truncated", false)
	show("/empty", false)
	show("/explicit", true)
}
`, strconv.Quote(server.URL))
	for name, data := range map[string]string{
		"main.go": program,
		"go.mod":  fmt.Sprintf("module httpgzipprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", strconv.Quote(root)),
	} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	native, err := testutil.Native(source, t.TempDir())
	if err != nil || native.Exit != 0 {
		t.Fatalf("native: %v\n%s", err, native)
	}
	want := testutil.Normalize(native)
	targets := []string{"go", "typescript", "python", "java", "csharp", "rust", "c", "swift"}
	if only := os.Getenv("GOALCHEMY_TEST_TARGETS"); only != "" {
		targets = strings.Split(only, ",")
	}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			out := t.TempDir()
			if ds := testutil.CompileGate(source, target, out, "cooperative"); len(ds) > 0 {
				t.Fatal(ds)
			}
			got, err := testutil.Runners[target](out)
			if err != nil {
				t.Fatal(err)
			}
			if g := testutil.Normalize(got); g != want {
				t.Errorf("%s differs from Go\n=== go\n%s=== %s\n%s", target, want, target, g)
			}
		})
	}
}
