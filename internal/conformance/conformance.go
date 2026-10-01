// Package conformance runs canonical contract cases against a target's
// runtime through its JSON Lines harness process.
package conformance

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"

	"goalchemy/internal/contracts"
)

type Result struct {
	Case   string
	Pass   bool
	Detail string
}

type request struct {
	V    int                        `json:"v"`
	ID   string                     `json:"id"`
	Case string                     `json:"case"`
	Let  map[string]json.RawMessage `json:"let"`
}

type response struct {
	V       int                        `json:"v"`
	ID      string                     `json:"id"`
	Status  string                     `json:"status"`
	Results []json.RawMessage          `json:"results"`
	After   map[string]json.RawMessage `json:"after"`
	Panic   string                     `json:"panic"`
	Error   string                     `json:"error"`
}

// Harness is a running target harness process.
type Harness struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Scanner
	n   int
}

func Start(root string, t *contracts.Target) (*Harness, error) {
	if t.Harness == nil {
		return nil, fmt.Errorf("target %s has no harness", t.Target)
	}
	cmd := exec.Command(t.Harness.Command[0], t.Harness.Command[1:]...)
	cmd.Dir = root
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	outp, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = &prefixWriter{}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	sc := bufio.NewScanner(outp)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	return &Harness{cmd: cmd, in: in, out: sc}, nil
}

type prefixWriter struct{ buf []byte }

func (p *prefixWriter) Write(b []byte) (int, error) { p.buf = append(p.buf, b...); return len(b), nil }

func (h *Harness) Close() error {
	h.in.Close()
	return h.cmd.Wait()
}

func (h *Harness) call(req request) (response, error) {
	h.n++
	req.V, req.ID = 1, fmt.Sprintf("r%d", h.n)
	data, _ := json.Marshal(req)
	if _, err := h.in.Write(append(data, '\n')); err != nil {
		return response{}, fmt.Errorf("process failure: %v: %s", err, h.cmd.Stderr.(*prefixWriter).buf)
	}
	if !h.out.Scan() {
		return response{}, fmt.Errorf("process failure: harness exited: %s", h.cmd.Stderr.(*prefixWriter).buf)
	}
	var resp response
	if err := json.Unmarshal(h.out.Bytes(), &resp); err != nil {
		return response{}, fmt.Errorf("protocol failure: %v", err)
	}
	if resp.ID != req.ID {
		return resp, fmt.Errorf("protocol failure: response %s for request %s", resp.ID, req.ID)
	}
	return resp, nil
}

// Run executes every case of every function the target implements.
func Run(cat *contracts.Catalog, root, target string) ([]Result, error) {
	t, ok := cat.Targets[target]
	if !ok {
		return nil, fmt.Errorf("unknown target %s", target)
	}
	h, err := Start(filepath.Clean(root), t)
	if err != nil {
		return nil, err
	}
	defer h.Close()
	var results []Result
	ids := make([]string, 0, len(t.Functions))
	for _, f := range t.Functions {
		if f.Harness != "" {
			ids = append(ids, f.ID)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		fc := cat.Functions[id]
		for _, c := range fc.Cases {
			r := runCase(h, fc, c)
			results = append(results, r)
		}
	}
	return results, nil
}

func runCase(h *Harness, fc *contracts.Function, c contracts.Case) Result {
	name := fc.ID + "/" + c.Name
	res := Result{Case: name}
	req := request{Case: name, Let: map[string]json.RawMessage{}}
	scope := contracts.Scope{}
	args := map[string]*contracts.TypeExpr{}
	for k, v := range c.Types {
		args[k], _ = contracts.ParseType(v)
	}
	for _, b := range c.Let {
		req.Let[b.Name] = b.Value
		te, _ := contracts.ParseType(b.Type)
		scope[b.Name] = te.Substitute(args)
	}
	resp, err := h.call(req)
	if err != nil {
		res.Detail = err.Error()
		return res
	}
	switch {
	case resp.Status == "harness_failure":
		res.Detail = "harness failure: " + resp.Error
		return res
	case c.Expect.Blocked:
		if resp.Status != "blocked" {
			res.Detail = "want blocked, got " + resp.Status
		} else {
			res.Pass = true
		}
		return res
	case c.Expect.Panic != nil:
		if resp.Status != "panic" {
			res.Detail = fmt.Sprintf("want panic %q, got %s", c.Expect.Panic.Message, resp.Status)
		} else if resp.Panic != c.Expect.Panic.Message {
			res.Detail = fmt.Sprintf("want panic %q, got panic %q", c.Expect.Panic.Message, resp.Panic)
		} else {
			res.Pass = true
		}
		return res
	case resp.Status != "returned":
		res.Detail = fmt.Sprintf("want return, got %s %s", resp.Status, resp.Panic)
		return res
	}
	if len(resp.Results) != len(c.Expect.Results) {
		res.Detail = fmt.Sprintf("want %d results, got %d", len(c.Expect.Results), len(resp.Results))
		return res
	}
	for i, raw := range c.Expect.Results {
		te, _ := contracts.ParseType(fc.Signature.Outputs[i].Type)
		if msg := compare(raw, te.Substitute(args), scope, resp.Results[i]); msg != "" {
			res.Detail = fmt.Sprintf("result %d: %s", i, msg)
			return res
		}
	}
	for _, a := range c.Expect.After {
		got, ok := resp.After[a.Name]
		if !ok {
			res.Detail = "missing postcondition " + a.Name
			return res
		}
		if msg := compare(a.Value, scope[a.Name], scope, got); msg != "" {
			res.Detail = fmt.Sprintf("after %s: %s", a.Name, msg)
			return res
		}
	}
	res.Pass = true
	return res
}

func compare(expected json.RawMessage, t *contracts.TypeExpr, scope contracts.Scope, actual json.RawMessage) string {
	v, err := contracts.DecodeValue(expected, t, scope)
	if err != nil {
		return "bad expectation: " + err.Error()
	}
	want := roundTrip(v.Encode())
	var got any
	if err := json.Unmarshal(actual, &got); err != nil {
		return "bad actual value"
	}
	got = stripCaps(want, got)
	if !reflect.DeepEqual(want, got) {
		w, _ := json.Marshal(want)
		g, _ := json.Marshal(got)
		return fmt.Sprintf("want %s, got %s", w, g)
	}
	return ""
}

func roundTrip(v any) any {
	data, _ := json.Marshal(v)
	var out any
	_ = json.Unmarshal(data, &out)
	return out
}

// stripCaps removes slice capacities the expectation does not mention.
func stripCaps(want, got any) any {
	wm, ok1 := want.(map[string]any)
	gm, ok2 := got.(map[string]any)
	if !ok1 || !ok2 {
		return got
	}
	out := map[string]any{}
	for k, v := range gm {
		out[k] = v
	}
	if _, ok := wm["slice"]; ok {
		if _, hasCap := wm["cap"]; !hasCap {
			delete(out, "cap")
		}
		ws, _ := wm["slice"].([]any)
		gs, _ := gm["slice"].([]any)
		if len(ws) == len(gs) {
			items := make([]any, len(gs))
			for i := range gs {
				items[i] = stripCaps(ws[i], gs[i])
			}
			out["slice"] = items
		}
	}
	return out
}
