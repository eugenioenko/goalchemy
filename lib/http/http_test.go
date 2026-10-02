package http

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExchangeAndRedirect(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path == "/redirect" {
			w.Header().Set("Location", "/secret")
			w.WriteHeader(302)
			return
		}
		if r.Method != "POST" || r.Header.Get("Authorization") != "Bearer test" || len(r.Header.Values("X-Request")) != 2 {
			t.Error("request", r)
		}
		b, _ := io.ReadAll(r.Body)
		if string(b) != "binary\x00" {
			t.Error("body")
		}
		w.Header().Add("X-Test", "a")
		w.Header().Add("X-Test", "b")
		w.Header().Set("DPoP-Nonce", "nonce")
		w.WriteHeader(403)
		w.Write([]byte("denied"))
	}))
	defer server.Close()
	status, h, b, e := Do(context.Background(), "POST", server.URL, []string{"Authorization", "Bearer test", "X-Request", "a", "X-Request", "b"}, []byte("binary\x00"), 6, 1000)
	if e != nil || status != 403 || string(b) != "denied" {
		t.Fatal(status, b, e)
	}
	count := 0
	nonce := false
	for i := 0; i < len(h); i += 2 {
		if h[i] == "X-Test" {
			count++
		}
		if strings.EqualFold(h[i], "DPoP-Nonce") && h[i+1] == "nonce" {
			nonce = true
		}
	}
	if count != 2 || !nonce {
		t.Fatal(h)
	}
	status, _, _, e = Do(context.Background(), "GET", server.URL+"/redirect", []string{"Authorization", "Bearer test"}, nil, 100, 1000)
	if e != nil || status != 302 || calls != 2 {
		t.Fatal(status, calls, e)
	}
}
func TestLimitsAndValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("12345")) }))
	defer server.Close()
	status, h, b, e := Do(context.Background(), "GET", server.URL, nil, nil, 4, 1000)
	if e == nil || status != 0 || h != nil || b != nil {
		t.Fatal("partial response", status, h, b, e)
	}
	for _, h := range [][]string{{"Odd"}, {"X\nBad", "v"}, {"X", "a\r\nb"}, {"Host", "evil"}, {"Content-Length", "1"}, {"X", "\x00"}} {
		if _, _, _, e := Do(context.Background(), "GET", server.URL, h, nil, 5, 1000); e == nil {
			t.Fatal(h)
		}
	}
	for _, url := range []string{"file:///tmp/a", "http://u:p@example.com", "http://example.com/#fragment"} {
		if _, _, _, e := Do(context.Background(), "GET", url, nil, nil, 5, 1000); e == nil {
			t.Fatal(url)
		}
	}
	if _, _, _, e := Do(context.Background(), "DELETE", server.URL, nil, nil, 5, 1000); e == nil {
		t.Fatal("method")
	}
	if _, _, _, e := Do(context.Background(), "GET", server.URL, nil, nil, 5, 0); e == nil {
		t.Fatal("unbounded time")
	}
	if _, _, _, e := Do(nil, "GET", server.URL, nil, nil, 5, 1000); e == nil {
		t.Fatal("nil context")
	}
}
func TestCancellationAndDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(25*time.Millisecond, cancel)
	defer timer.Stop()
	start := time.Now()
	if _, _, _, e := Do(ctx, "GET", server.URL, nil, nil, 10, 1000); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if time.Since(start) > time.Second {
		t.Fatal("cancellation delayed")
	}
	if _, _, _, e := Do(context.Background(), "GET", server.URL, nil, nil, 10, 25); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal(e)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if _, _, _, e := Do(ctx, "GET", server.URL, nil, nil, 10, 1000); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal(e)
	}
}
func TestTLSVerificationAndHeaderBound(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("untrusted TLS reached handler") }))
	defer server.Close()
	if _, _, _, e := Do(context.Background(), "GET", server.URL, nil, nil, 5, 1000); e == nil {
		t.Fatal("untrusted TLS accepted")
	}
	large := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Large", strings.Repeat("x", MaxHeaderBytes+1))
		w.WriteHeader(200)
	}))
	defer large.Close()
	if _, _, _, e := Do(context.Background(), "GET", large.URL, nil, nil, 5, 1000); e == nil {
		t.Fatal("large reply headers")
	}
}

func TestDecodedBodyLimitAndGETBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		g := gzip.NewWriter(w)
		g.Write([]byte(strings.Repeat("x", 1000)))
		g.Close()
	}))
	defer server.Close()
	if _, _, b, e := Do(context.Background(), "GET", server.URL, nil, nil, 999, 1000); e == nil || b != nil {
		t.Fatal("decoded bound", e)
	}
	status, h, b, e := Do(context.Background(), "GET", server.URL, nil, nil, 1000, 1000)
	if e != nil || status != 200 || len(b) != 1000 {
		t.Fatal(status, len(b), e)
	}
	for i := 0; i < len(h); i += 2 {
		if h[i] == "Content-Encoding" {
			t.Fatal("decoded encoding header")
		}
	}
	if _, _, _, e := Do(context.Background(), "GET", server.URL, nil, []byte("body"), 1000, 1000); e == nil {
		t.Fatal("GET body")
	}
}
