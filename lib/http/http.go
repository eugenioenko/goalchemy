// SPDX-License-Identifier: Apache-2.0
// Package http provides bounded HTTP exchanges, retaining TLS verification
// and returning redirect responses without following them.
package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	stdhttp "net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// MaxBytes is the largest request or response body Do accepts (64MiB).
const MaxBytes = 64 << 20

// MaxHeaderBytes is the largest request or response header size Do accepts (64KiB).
const MaxHeaderBytes = 64 << 10

var invalid = errors.New("http: invalid request or limit")
var tooLarge = errors.New("http: response body exceeds limit")

// Do performs one bounded HTTP GET or POST exchange. It takes alternating header names/values. timeoutMillis uses real elapsed
// host time; it is required (1..300000). A parent deadline can shorten it.
// Result headers preserve duplicate values, with names sorted, canonicalized.
// Non-2xx/redirect statuses are ordinary results; transport/limit failures
// return zero status and nil headers/body, never a partial successful body.
func Do(ctx context.Context, method, rawURL string, headers []string, body []byte, maxResponseBytes int, timeoutMillis int64) (int, []string, []byte, error) {
	if ctx == nil || (method != "GET" && method != "POST") || len(body) > MaxBytes || (method == "GET" && len(body) != 0) || maxResponseBytes < 0 || maxResponseBytes > MaxBytes || timeoutMillis < 1 || timeoutMillis > 300000 || len(headers)%2 != 0 || len(rawURL) > 8192 {
		return 0, nil, nil, invalid
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Fragment != "" {
		return 0, nil, nil, invalid
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMillis)*time.Millisecond)
	defer cancel()
	req, err := stdhttp.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
	if err != nil {
		return 0, nil, nil, invalid
	}
	total := 0
	for i := 0; i < len(headers); i += 2 {
		name, value := headers[i], headers[i+1]
		total += len(name) + len(value) + 4
		if total > MaxHeaderBytes || !token(name) || badValue(value) || forbidden(name) {
			return 0, nil, nil, invalid
		}
		req.Header.Add(name, value)
	}
	// Dedicated transport avoids unbounded global pooled resources and host
	// default-client mutations; proxies follow the standard trusted host config.
	tr := &stdhttp.Transport{Proxy: stdhttp.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: 30 * time.Second}).DialContext, TLSHandshakeTimeout: 10 * time.Second, DisableKeepAlives: true, MaxResponseHeaderBytes: MaxHeaderBytes}
	defer tr.CloseIdleConnections()
	client := &stdhttp.Client{Transport: tr, CheckRedirect: func(*stdhttp.Request, []*stdhttp.Request) error { return stdhttp.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxResponseBytes)+1))
	if err != nil {
		return 0, nil, nil, err
	}
	if len(out) > maxResponseBytes {
		return 0, nil, nil, tooLarge
	}
	names := make([]string, 0, len(resp.Header))
	for name := range resp.Header {
		names = append(names, name)
	}
	sort.Strings(names)
	var result []string
	for _, name := range names {
		for _, value := range resp.Header[name] {
			result = append(result, name, value)
		}
	}
	return resp.StatusCode, result, out, nil
}
func token(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", c)) {
			return false
		}
	}
	return true
}
func forbidden(s string) bool {
	switch strings.ToLower(s) {
	case "host", "content-length", "transfer-encoding", "connection", "proxy-authorization", "proxy-connection", "upgrade", "trailer", "te":
		return true
	}
	return false
}

func badValue(s string) bool {
	for _, c := range s {
		if c == 127 || c < 32 && c != '\t' {
			return true
		}
	}
	return false
}
