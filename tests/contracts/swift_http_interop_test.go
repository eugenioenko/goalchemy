package contracts

import (
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eugenioenko/goalchemy/internal/testutil"
)

func TestSwiftHTTPInterop(t *testing.T) {
	var redirects atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/headers":
			w.Header().Add("X-Multi", "first")
			w.Header().Add("X-Multi", "second")
			w.Header().Add("Set-Cookie", "a=1; Expires=Wed, 09 Jun 2027 10:18:14 GMT")
			w.Header().Add("Set-Cookie", "b=2")
			w.Write([]byte{0, 255, 128})
		case "/post":
			if r.Method != "POST" || r.Header.Get("X-Probe") != "kept" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			io.Copy(w, r.Body)
		case "/gzip":
			w.Header().Set("Content-Encoding", "gzip")
			z := gzip.NewWriter(w)
			z.Write([]byte{0, 255, 128})
			z.Close()
		case "/large":
			io.WriteString(w, strings.Repeat("x", 64<<10))
		case "/redirect":
			w.Header().Set("Location", "/followed")
			w.WriteHeader(http.StatusFound)
		case "/followed":
			redirects.Add(1)
			io.WriteString(w, "redirect followed")
		case "/slow":
			select {
			case <-r.Context().Done():
			case <-time.After(250 * time.Millisecond):
				io.WriteString(w, "too late")
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	tls := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "untrusted")
	}))
	defer tls.Close()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	source, out := t.TempDir(), t.TempDir()
	program := fmt.Sprintf(`package main
import (
 "github.com/eugenioenko/goalchemy/lib/context"
 "github.com/eugenioenko/goalchemy/lib/errors"
 "github.com/eugenioenko/goalchemy/lib/http"
 "github.com/eugenioenko/goalchemy/lib/time"
 "github.com/eugenioenko/goalchemy/std/encoding/hex"
)
const base = %s
const untrusted = %s
func need(err error) { if err != nil { panic(err.Error()) } }
func failed(status int, headers []string, body []byte, err error) {
 if err == nil || status != 0 || headers != nil || body != nil { panic("invalid failure outputs") }
}
func main() {
 ctx := context.Background()
 status, headers, body, err := http.Do(ctx, "GET", base+"/headers", nil, nil, 1024, 1000)
 need(err); if status != 200 { panic("status") }
 println("BODY", hex.EncodeToString(body))
 last := ""
 for i:=0; i<len(headers); i+=2 {
   if headers[i] < last { panic("unsorted headers") }; last=headers[i]
   if headers[i] == "X-Multi" || headers[i] == "Set-Cookie" {
     println("HEADER", headers[i], hex.EncodeToString([]byte(headers[i+1])))
   }
 }
 sent := []byte{0,255,128}
 status, _, body, err = http.Do(ctx,"POST",base+"/post",[]string{"X-Probe","kept"},sent,1024,1000)
 need(err); if status!=200 || sent[0]!=0 { panic("POST") }; println("POST",hex.EncodeToString(body))
 status, headers, body, err = http.Do(ctx,"GET",base+"/gzip",nil,nil,1024,1000)
 need(err); if status!=200 { panic("gzip status") }
 for i:=0;i<len(headers);i+=2 { if headers[i]=="Content-Encoding" || headers[i]=="Content-Length" { panic("decoded framing headers retained") } }
 println("GZIP",hex.EncodeToString(body))
 status, headers, body, err = http.Do(ctx,"GET",base+"/large",nil,nil,16,1000)
 failed(status,headers,body,err)
 status, _, _, err = http.Do(ctx,"GET",base+"/redirect",nil,nil,1024,1000)
 need(err); if status!=302 { panic("redirect followed") }
 status, headers, body, err = http.Do(ctx,"GET",base+"/slow",nil,nil,1024,10)
 failed(status,headers,body,err)
 child,cancel := context.WithCancel(ctx)
 go func() { time.Sleep(5*time.Millisecond); cancel() }()
 status, headers, body, err = http.Do(child,"GET",base+"/slow",nil,nil,1024,1000)
 failed(status,headers,body,err)
 if !errors.Is(err,context.Canceled) { panic("cancellation sentinel") }
 deadline,stop := context.WithTimeout(ctx,5*time.Millisecond)
 status, headers, body, err = http.Do(deadline,"GET",base+"/slow",nil,nil,1024,1000)
 stop(); failed(status,headers,body,err)
 if !errors.Is(err,context.DeadlineExceeded) { panic("deadline sentinel") }
 status, headers, body, err = http.Do(ctx,"GET",base+"/headers",[]string{"Content-Length","1"},nil,1024,1000)
 failed(status,headers,body,err)
 status, headers, body, err = http.Do(ctx,"GET",untrusted,nil,nil,1024,1000)
 failed(status,headers,body,err)
 println("CHECK","http-ok")
}
`, strconv.Quote(server.URL), strconv.Quote(tls.URL))
	for name, data := range map[string]string{
		"main.go": program,
		"go.mod":  fmt.Sprintf("module swifthttpprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => %s\n", strconv.Quote(root)),
	} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if ds := testutil.CompileGate(source, "swift", out, "cooperative"); len(ds) > 0 {
		t.Fatal(ds)
	}
	got, err := testutil.Runners["swift"](out)
	if err != nil || got.Exit != 0 {
		t.Fatalf("Swift HTTP: %v\n%s", err, got)
	}
	if !strings.Contains(got.Stderr, "CHECK http-ok") {
		t.Fatalf("HTTP checks did not finish:\n%s", got)
	}
	expectedBody := hex.EncodeToString([]byte{0, 255, 128})
	for _, label := range []string{"BODY", "POST", "GZIP"} {
		if !strings.Contains(got.Stderr, label+" "+expectedBody+"\n") {
			t.Errorf("missing %s bytes: %s", label, got.Stderr)
		}
	}
	var cookies, multi []string
	for _, line := range strings.Split(got.Stderr, "\n") {
		parts := strings.Fields(line)
		if len(parts) != 3 || parts[0] != "HEADER" {
			continue
		}
		value, err := hex.DecodeString(parts[2])
		if err != nil {
			t.Fatal(err)
		}
		if parts[1] == "Set-Cookie" {
			cookies = append(cookies, string(value))
		} else if parts[1] == "X-Multi" {
			multi = append(multi, string(value))
		}
	}
	if strings.Join(multi, "\n") != "first\nsecond" {
		t.Errorf("duplicate header values changed: %q", multi)
	}
	if strings.Join(cookies, "\n") != "a=1; Expires=Wed, 09 Jun 2027 10:18:14 GMT\nb=2" {
		t.Errorf("duplicate cookies changed: %q", cookies)
	}
	if redirects.Load() != 0 {
		t.Error("native HTTP followed a redirect")
	}
}
