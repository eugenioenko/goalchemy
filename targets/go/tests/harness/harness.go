// Command harness serves runtime conformance requests for the Go target
// over JSON Lines on standard input and output.
package main

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"

	rt "goalchemy/targets/go/runtime"
)

const protocolVersion = 1

type request struct {
	V    int                        `json:"v"`
	ID   string                     `json:"id"`
	Case string                     `json:"case"`
	Let  map[string]json.RawMessage `json:"let"`
}

type response struct {
	V       int            `json:"v"`
	ID      string         `json:"id"`
	Status  string         `json:"status"`
	Results []any          `json:"results,omitempty"`
	After   map[string]any `json:"after,omitempty"`
	Panic   string         `json:"panic,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// H gives a case access to its typed inputs and postcondition outputs.
type H struct {
	let   map[string]json.RawMessage
	after map[string]any
}

func (h *H) Let(name string) json.RawMessage { return h.let[name] }
func (h *H) After(name string, v any)       { h.after[name] = v }

type harnessError struct{ msg string }

// blockedSignal reports that the case's task blocked with nothing runnable.
type blockedSignal struct{}

func fail(format string, args ...any) { panic(harnessError{fmt.Sprintf(format, args...)}) }

func main() {
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 1<<20), 1<<26)
	out := bufio.NewWriter(os.Stdout)
	enc := json.NewEncoder(out)
	for in.Scan() {
		var req request
		resp := response{V: protocolVersion}
		if err := json.Unmarshal(in.Bytes(), &req); err != nil {
			resp.Status, resp.Error = "harness_failure", err.Error()
		} else {
			resp = serve(req)
		}
		_ = enc.Encode(resp)
		_ = out.Flush()
	}
}

func serve(req request) (resp response) {
	resp = response{V: protocolVersion, ID: req.ID}
	if req.V != protocolVersion {
		resp.Status, resp.Error = "harness_failure", "unsupported protocol version"
		return
	}
	fn, ok := cases[req.Case]
	if !ok {
		resp.Status, resp.Error = "harness_failure", "unknown case "+req.Case
		return
	}
	h := &H{let: req.Let, after: map[string]any{}}
	rt.ResetScheduler(func() { panic(blockedSignal{}) })
	defer func() {
		if r := recover(); r != nil {
			if he, ok := r.(harnessError); ok {
				resp.Status, resp.Error = "harness_failure", he.msg
				return
			}
			if _, ok := r.(blockedSignal); ok {
				resp.Status = "blocked"
				return
			}
			resp.Status = "panic"
			switch x := r.(type) {
			case error:
				resp.Panic = x.Error()
			case string:
				resp.Panic = x
			default:
				resp.Panic = fmt.Sprint(x)
			}
		}
	}()
	resp.Results = fn(h)
	if resp.Results == nil {
		resp.Results = []any{}
	}
	resp.After = h.after
	resp.Status = "returned"
	return
}

func object(raw json.RawMessage) map[string]json.RawMessage {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		fail("expected object, got %s", raw)
	}
	return m
}

func str(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		fail("expected string, got %s", raw)
	}
	return s
}

func DecInt[T rt.Integer](raw json.RawMessage) T {
	n, ok := new(big.Int).SetString(str(raw), 10)
	if !ok {
		fail("bad integer %s", raw)
	}
	var zero T
	if zero-1 < 0 {
		return T(n.Int64())
	}
	return T(n.Uint64())
}

func DecError(raw json.RawMessage) error {
	m := object(raw)
	if _, ok := m["nil"]; ok {
		return nil
	}
	return rt.StdErrorsNew(str(m["error"]))
}

func EncError(err error) any {
	if err == nil {
		return map[string]any{"nil": true}
	}
	return map[string]any{"error": err.Error()}
}

func DecBool(raw json.RawMessage) bool { return str(raw) == "true" }

func DecString(raw json.RawMessage) string {
	m := object(raw)
	if s, ok := m["str"]; ok {
		return str(s)
	}
	b, err := hex.DecodeString(str(m["hex"]))
	if err != nil {
		fail("bad hex string")
	}
	return string(b)
}

func DecSlice[T any](raw json.RawMessage, dec func(json.RawMessage) T) []T {
	m := object(raw)
	if _, ok := m["nil"]; ok {
		return nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(m["slice"], &items); err != nil {
		fail("bad slice")
	}
	c := len(items)
	if cr, ok := m["cap"]; ok {
		c, _ = strconv.Atoi(str(cr))
	}
	s := make([]T, len(items), c)
	for i, it := range items {
		s[i] = dec(it)
	}
	return s
}

func View[T any](base []T, raw json.RawMessage) []T {
	m := object(raw)
	lo, hi, mx := 0, len(base), cap(base)
	if v, ok := m["lo"]; ok {
		lo, _ = strconv.Atoi(str(v))
	}
	if v, ok := m["hi"]; ok {
		hi, _ = strconv.Atoi(str(v))
	}
	if v, ok := m["max"]; ok {
		mx, _ = strconv.Atoi(str(v))
	}
	return base[lo:hi:mx]
}

func DecMap[K comparable, V any](raw json.RawMessage, dk func(json.RawMessage) K, dv func(json.RawMessage) V) rt.Map[K, V] {
	m := object(raw)
	if _, ok := m["nil"]; ok {
		return rt.Map[K, V]{}
	}
	var entries []struct {
		Key   json.RawMessage `json:"key"`
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(m["map"], &entries); err != nil {
		fail("bad map")
	}
	out := rt.NewMap[K, V]()
	for _, e := range entries {
		out.Set(dk(e.Key), dv(e.Value))
	}
	return out
}

func EncInt[T rt.Integer](v T) any {
	var zero T
	if zero-1 < 0 {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatUint(uint64(v), 10)
}

func EncBool(v bool) any { return strconv.FormatBool(v) }

func EncString(v string) any { return map[string]any{"hex": hex.EncodeToString([]byte(v))} }

func EncSlice[T any](s []T, enc func(T) any) any {
	if s == nil {
		return map[string]any{"nil": true}
	}
	items := make([]any, len(s))
	for i, e := range s {
		items[i] = enc(e)
	}
	return map[string]any{"slice": items, "cap": strconv.Itoa(cap(s))}
}

func EncMap[K comparable, V any](m rt.Map[K, V], ek func(K) any, ev func(V) any) any {
	if m.IsNil() {
		return map[string]any{"nil": true}
	}
	items := []any{}
	it := m.Iter()
	for {
		ok, k, v := it.Next()
		if !ok {
			break
		}
		items = append(items, map[string]any{"key": ek(k), "value": ev(v)})
	}
	return map[string]any{"map": items}
}

func EncArray[T any](s []T, enc func(T) any) any {
	items := make([]any, len(s))
	for i, e := range s {
		items[i] = enc(e)
	}
	return map[string]any{"array": items}
}

func DecChan[T any](raw json.RawMessage, dec func(json.RawMessage) T) rt.Chan[T] {
	m := object(raw)
	if _, ok := m["nil"]; ok {
		return rt.Chan[T]{}
	}
	var items []json.RawMessage
	if err := json.Unmarshal(m["chan"], &items); err != nil {
		fail("bad channel")
	}
	size := 0
	if c, ok := m["cap"]; ok {
		size, _ = strconv.Atoi(str(c))
	}
	ch := rt.MakeChan[T](size)
	for _, it := range items {
		ch.Send(dec(it))
	}
	if c, ok := m["closed"]; ok && string(c) == "true" {
		ch.Close()
	}
	return ch
}

func EncChan[T any](ch rt.Chan[T], enc func(T) any) any {
	if ch == (rt.Chan[T]{}) {
		return map[string]any{"nil": true}
	}
	items := []any{}
	closed := rt.ChanClosed(ch)
	for _, v := range rt.ChanBuffered(ch) {
		items = append(items, enc(v))
	}
	return map[string]any{"chan": items, "cap": strconv.Itoa(ch.Cap()), "closed": closed}
}

func EncZero(any) any { return map[string]any{"zero": true} }

// HarnessSpawn reports whether a spawned task ran before the parent yielded.
func HarnessSpawn() bool {
	ran := false
	rt.Go(func() { ran = true })
	before := ran
	rt.Gosched()
	if !ran {
		fail("spawned task did not run after the parent yielded")
	}
	return before
}

// HarnessSelect2 selects over receives from a and b.
func HarnessSelect2[T any](a, b rt.Chan[T], dflt bool) int {
	i, _, _ := rt.Select([]rt.SelectCase{rt.RecvCase(a), rt.RecvCase(b)}, dflt)
	return i
}

func HarnessLockUnlock(m *rt.Mutex) {
	rt.StdSyncMutexLock(m)
	rt.StdSyncMutexUnlock(m)
}
