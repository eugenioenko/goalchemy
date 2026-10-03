package integration

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eugenioenko/goalchemy/internal/testutil"
)

func emittedHTTP(t *testing.T, source string) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	fixture, out := t.TempDir(), t.TempDir()
	mod := fmt.Sprintf("module hostprobe\n\ngo 1.25\n\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", root)
	if err := os.WriteFile(filepath.Join(fixture, "go.mod"), []byte(mod), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "main.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	if ds := testutil.CompileGate(fixture, "go", out, "cooperative"); len(ds) > 0 {
		t.Fatal(ds)
	}
	main, err := os.ReadFile(filepath.Join(out, "main.go"))
	if err != nil || !strings.Contains(string(main), "rt.RunMainHost(") {
		t.Fatalf("HTTP did not select explicit host entry: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	// Toolchain downloads and compiler diagnostics are not source observations.
	// Execute the race-enabled binary separately, keeping its output assertion exact.
	binary := filepath.Join(out, "host-probe")
	build := exec.CommandContext(ctx, "go", "build", "-race", "-o", binary, ".")
	build.Dir = out
	build.Env = append(os.Environ(), "GOTOOLCHAIN=go1.25.14", "GOFLAGS=")
	if diagnostics, err := build.CombinedOutput(); err != nil {
		t.Fatalf("emitted HTTP build failed: %v\n%s", err, diagnostics)
	} else if len(diagnostics) != 0 {
		t.Logf("emitted HTTP build diagnostics:\n%s", diagnostics)
	}
	cmd := exec.CommandContext(ctx, binary)
	cmd.Dir, cmd.Env = out, build.Env
	result, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("actual emitted HTTP failed: %v\n%s", err, result)
	}
	return string(result)
}

const httpProbeImports = `package main
import (
 "github.com/eugenioenko/goalchemy/lib/context"
 "github.com/eugenioenko/goalchemy/lib/http"
 "github.com/eugenioenko/goalchemy/lib/time"
 "github.com/eugenioenko/goalchemy/lib/runtime"
 "github.com/eugenioenko/goalchemy/lib/errors"
 "github.com/eugenioenko/goalchemy/lib/clock"
)
`

func TestGeneratedGoHTTPHostLifecycle(t *testing.T) {
	headersStarted, bodyStarted := make(chan struct{}), make(chan struct{})
	shutdownStarted, shutdownCanceled := make(chan struct{}), make(chan struct{})
	echoRelease := make(chan struct{})
	var forbiddenRequests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/headers":
			close(headersStarted)
			<-r.Context().Done()
		case "/headersStarted":
			select {
			case <-headersStarted:
				w.Write([]byte("started"))
			case <-r.Context().Done():
			}
		case "/body":
			w.Write([]byte("partial"))
			w.(http.Flusher).Flush()
			close(bodyStarted)
			<-r.Context().Done()
		case "/bodyStarted":
			select {
			case <-bodyStarted:
				w.Write([]byte("started"))
			case <-r.Context().Done():
			}
		case "/timeout", "/parent":
			w.Write([]byte("partial"))
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		case "/forbidden":
			forbiddenRequests.Add(1)
			w.Write([]byte("wrong"))
		case "/echo":
			b, _ := io.ReadAll(r.Body)
			<-echoRelease
			w.Header().Add("X-Duplicate", "one")
			w.Header().Add("X-Duplicate", "two")
			w.Write([]byte(r.Header.Get("X-Input") + ":" + string(b)))
		case "/releaseEcho":
			close(echoRelease)
			w.Write([]byte("released"))
		case "/shutdown":
			close(shutdownStarted)
			<-r.Context().Done()
			close(shutdownCanceled)
		case "/shutdownStarted":
			select {
			case <-shutdownStarted:
				w.Write([]byte("started"))
			case <-r.Context().Done():
			}
		case "/ok":
			w.Write([]byte("ok"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	// All waits proving source cancellation are synchronized with server request
	// state, rather than assuming a source sleep was long enough to start I/O.
	source := httpProbeImports + fmt.Sprintf(`
func must(ok bool) { if !ok { panic("host lifecycle assertion") } }
func control(path string) {
 s, _, b, e := http.Do(context.Background(), "GET", %q+path, nil, nil, 64, 2000)
 must(e == nil && s == 200 && len(b) > 0)
}
func canceled(path string, started string) {
 ctx, cancel := context.WithCancel(context.Background())
 done := make(chan bool)
 go func() {
  s, h, b, e := http.Do(ctx, "GET", %q+path, nil, nil, 64, 2000)
  done <- s == 0 && h == nil && b == nil && errors.Is(e, context.Canceled)
 }()
 control(started)
 cancel()
 must(<-done)
 must(ctx.Err() == context.Canceled)
 <-ctx.Done()
}
// Function-value dispatch must still select host entry and carry suspension.
func exchange(ctx context.Context, path string, limit int, timeout int64) (int, []string, []byte, error) {
 return http.Do(ctx, "GET", %q+path, nil, nil, limit, timeout)
}
func main() {
 canceled("/headers", "/headersStarted")
 canceled("/body", "/bodyStarted")
 f := exchange
 s, h, b, e := f(context.Background(), "/timeout", 64, 25)
 must(s == 0 && h == nil && b == nil && errors.Is(e, context.DeadlineExceeded))
 parent, pcancel := context.WithTimeout(context.Background(), 30000000)
 child, ccancel := context.WithTimeout(parent, 2000000000)
 s, h, b, e = f(child, "/parent", 64, 2000)
 must(s == 0 && h == nil && b == nil && errors.Is(e, context.DeadlineExceeded))
 must(parent.Err() == context.DeadlineExceeded && child.Err() == context.DeadlineExceeded)
 pcancel(); ccancel()
 // A timeout begins before HTTP submission. The source sleeper uses host time
 // even when no HTTP operation is pending, so this parent has expired already.
 expired, ecancel := context.WithTimeout(context.Background(), 10000000)
 unixBefore := clock.Unix()
 time.Sleep(30000000)
 must(clock.Unix() >= unixBefore)
 s, h, b, e = f(expired, "/forbidden", 64, 2000)
 must(s == 0 && h == nil && b == nil && errors.Is(e, context.DeadlineExceeded))
 ecancel()
 already, acancel := context.WithCancel(context.Background()); acancel()
 s, h, b, e = f(already, "/forbidden", 64, 2000)
 must(s == 0 && h == nil && b == nil && errors.Is(e, context.Canceled))
 // mutable inputs are copied before the submitting task yields
 input := []byte("abc"); inputHeaders := []string{"X-Input", "old"}
 echoDone := make(chan bool)
 go func() {
  es, eh, eb, ee := http.Do(context.Background(), "POST", %q+"/echo", inputHeaders, input, 64, 2000)
  duplicates := 0
  for i := 0; i < len(eh); i += 2 { if eh[i] == "X-Duplicate" { duplicates++ } }
  echoDone <- es == 200 && ee == nil && string(eb) == "old:abc" && duplicates == 2
  if len(eb) > 0 { eb[0] = 'Z' }; if len(eh) > 1 { eh[1] = "changed" }
 }()
 runtime.Gosched()
 input[0] = 'z'; inputHeaders[1] = "new"
 control("/releaseEcho")
 must(<-echoDone)
 s, _, b, e = f(context.Background(), "/ok", 64, 2000)
 must(e == nil && s == 200 && string(b) == "ok")
 println("host lifecycle ok")
 go func() { http.Do(context.Background(), "GET", %q+"/shutdown", nil, nil, 64, 2000) }()
 control("/shutdownStarted")
}
`, srv.URL, srv.URL, srv.URL, srv.URL, srv.URL)
	output := emittedHTTP(t, source)
	if output != "host lifecycle ok\n" {
		t.Fatalf("unexpected source output %q", output)
	}
	select {
	case <-shutdownCanceled:
	case <-time.After(5 * time.Second):
		t.Fatal("entry shutdown leaked background transport")
	}
	if forbiddenRequests.Load() != 0 {
		t.Fatal("expired/already-canceled request reached host server")
	}
}

func TestGeneratedGoHTTPNativeBoundary(t *testing.T) {
	var redirectFollowed atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			w.Header().Set("Location", "/follow")
			w.WriteHeader(302)
			w.Write([]byte("redirect"))
		case "/follow":
			redirectFollowed.Add(1)
			w.Write([]byte("wrong"))
		case "/non2xx":
			w.WriteHeader(418)
			w.Write([]byte("teapot"))
		case "/large":
			w.Write([]byte("12345"))
		case "/gzip":
			w.Header().Set("Content-Encoding", "gzip")
			z := gzip.NewWriter(w)
			z.Write([]byte("12345"))
			z.Close()
		case "/largeHeaders":
			w.Header().Set("X-Large", strings.Repeat("x", 70<<10))
			w.Write([]byte("x"))
		case "/empty":
			w.WriteHeader(204)
		default:
			w.Write([]byte("ok"))
		}
	}))
	defer srv.Close()
	tls := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("bad tls")) }))
	defer tls.Close()
	source := `package main
import (
 "github.com/eugenioenko/goalchemy/lib/context"
 "github.com/eugenioenko/goalchemy/lib/http"
)
func must(ok bool) { if !ok { panic("native boundary assertion") } }
` + fmt.Sprintf(`
func get(path string, limit int) (int, []string, []byte, error) {
 return http.Do(context.Background(), "GET", %q+path, nil, nil, limit, 2000)
}
func failure(method string, url string, headers []string, body []byte, limit int, timeout int64) {
 s, h, b, e := http.Do(context.Background(), method, url, headers, body, limit, timeout)
 must(s == 0 && h == nil && b == nil && e != nil)
}
func main() {
 s, _, b, e := get("/redirect", 64); must(s == 302 && e == nil && string(b) == "redirect")
 s, _, b, e = get("/non2xx", 64); must(s == 418 && e == nil && string(b) == "teapot")
 s, _, b, e = get("/empty", 0); must(s == 204 && e == nil && len(b) == 0)
 s, h, b, e := get("/large", 4); must(s == 0 && h == nil && b == nil && e != nil)
 s, h, b, e = get("/gzip", 4); must(s == 0 && h == nil && b == nil && e != nil)
 s, h, b, e = get("/largeHeaders", 64); must(s == 0 && h == nil && b == nil && e != nil)
 failure("GET", %q, nil, nil, 64, 2000)
 failure("PUT", %q, nil, nil, 64, 2000)
 failure("GET", %q, nil, []byte("body"), 64, 2000)
 failure("GET", %q, []string{"odd"}, nil, 64, 2000)
 failure("GET", %q, []string{"Host", "evil"}, nil, 64, 2000)
 failure("GET", %q, []string{"X-Test", "bad\nvalue"}, nil, 64, 2000)
 failure("GET", %q, nil, nil, 64, 0)
 failure("GET", %q, nil, nil, 64, 300001)
 failure("GET", %q+"#fragment", nil, nil, 64, 2000)
 var nilContext context.Context
 must(nilContext == nil && nil == nilContext && !(nilContext != nil) && !(nil != nilContext))
 type Alias = context.Context
 var alias Alias = nil
 must(alias == nil && nil == alias)
 live := context.Background()
 must(live != nil && nil != live)
 s, h, b, e = http.Do(nilContext, "GET", %q, nil, nil, 64, 2000)
 must(s == 0 && h == nil && b == nil && e != nil)
 println("native boundary ok")
}
`, srv.URL, tls.URL, srv.URL, srv.URL, srv.URL, srv.URL, srv.URL, srv.URL, srv.URL, srv.URL, srv.URL)
	if output := emittedHTTP(t, source); output != "native boundary ok\n" {
		t.Fatalf("unexpected output %q", output)
	}
	if redirectFollowed.Load() != 0 {
		t.Fatal("generated native mapping followed redirect")
	}
}
