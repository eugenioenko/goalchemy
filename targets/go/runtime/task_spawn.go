package rt

import (
	"os"
	"strconv"
)

// The cooperative scheduler. Suspending functions are compiled to resumable
// frames: a frame holds the function's locals and the block to resume at,
// and Step runs it until it returns or reaches a pause point. A task is a
// stack of frames driven by a trampoline; exactly one task runs at a time,
// and runnable tasks are dispatched in FIFO order. Pause primitives either
// complete immediately, leaving their results in Task.RV, or block the task
// until another task or a timer readies it. Deferred calls, panics, and
// recover are managed per task by the runtime, because frames do not live
// on the host stack.

// Frame is a resumable activation of a suspending function.
type Frame interface {
	Base() *FrameBase
	Step(t *Task)
	Results() []any
}

// FrameBase is embedded in every frame.
type FrameBase struct {
	PC        int
	Defers    []Deferred
	parent    Frame
	panicking *Panic
}

func (b *FrameBase) Base() *FrameBase { return b }

// Deferred is a deferred call with its callee and arguments evaluated:
// Call for ordinary callees, Start for suspending ones.
type Deferred struct {
	Call  func()
	Start func() Frame
	Fid   int
}

// Panic is a source panic in flight.
type Panic struct {
	Value     any
	Recovered bool
	Prev      *Panic
}

type Task struct {
	id          int
	Frame       Frame
	RV          []any
	blocked     bool
	done        bool
	resumePanic *Panic
	curPanic    *Panic
	deferTarget int
	// cleanup removes a blocked select's losing registrations on resume.
	cleanup func()
}

type timer struct {
	at  int64
	seq int64
	t   *Task
	fn  func()
}

type scheduler struct {
	runq    []*Task
	cur     *Task
	main    *Task
	nextID  int
	rng     uint32
	clock   int64
	timers  []*timer
	seq     int64
	harness bool
}

var sched = &scheduler{rng: 1}

func seed() uint32 {
	if s := os.Getenv("GOALCHEMY_SEED"); s != "" {
		if n, err := strconv.ParseUint(s, 10, 32); err == nil && n != 0 {
			return uint32(n)
		}
	}
	return 1
}

// choose returns a uniformly chosen index below n from the xorshift32
// choice source shared by every target.
func (s *scheduler) choose(n int) int {
	x := s.rng
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	s.rng = x
	return int(x % uint32(n))
}

func (s *scheduler) ready(t *Task) { s.runq = append(s.runq, t) }

// block parks the current task until something readies it.
func (s *scheduler) block(t *Task) { t.blocked = true }

type deadlockSignal struct{}

// fatalPanicSignal carries an unrecovered panic out of a harness case.
type fatalPanicSignal struct{ p *Panic }

func (s *scheduler) next() *Task {
	for len(s.runq) == 0 {
		if len(s.timers) == 0 {
			if s.harness {
				panic(deadlockSignal{})
			}
			fatal("all goroutines are asleep - deadlock!")
		}
		s.fireTimers()
	}
	t := s.runq[0]
	s.runq = s.runq[1:]
	return t
}

func (s *scheduler) fireTimers() {
	at := s.timers[0].at
	for _, t := range s.timers {
		at = min(at, t.at)
	}
	s.clock = at
	var due, keep []*timer
	for _, t := range s.timers {
		if t.at == at {
			due = append(due, t)
		} else {
			keep = append(keep, t)
		}
	}
	s.timers = keep
	for i := 1; i < len(due); i++ {
		for j := i; j > 0 && due[j].seq < due[j-1].seq; j-- {
			due[j], due[j-1] = due[j-1], due[j]
		}
	}
	for _, t := range due {
		if t.fn != nil {
			t.fn()
		}
		if t.t != nil {
			s.ready(t.t)
		}
	}
}

func (s *scheduler) addTimer(d int64, t *Task, fn func()) {
	s.seq++
	s.timers = append(s.timers, &timer{at: s.clock + d, seq: s.seq, t: t, fn: fn})
}

func fatal(msg string) {
	os.Stderr.WriteString("fatal error: " + msg + "\n")
	os.Exit(2)
}

// run drives a task's frames until it blocks or finishes.
func (s *scheduler) run(t *Task) {
	s.cur = t
	t.blocked = false
	if c := t.cleanup; c != nil {
		t.cleanup = nil
		c()
	}
	for !t.blocked && t.Frame != nil {
		if p := t.resumePanic; p != nil {
			t.resumePanic = nil
			s.exit(t, t.Frame, p)
			continue
		}
		s.step(t)
	}
}

func (s *scheduler) step(t *Task) {
	f := t.Frame
	defer func() {
		if r := recover(); r != nil {
			s.exit(t, f, &Panic{Value: r})
		}
	}()
	f.Step(t)
}

// exit leaves frame f, with p non-nil when it is panicking. Deferred calls
// run first, through a deferRunner frame pushed above f.
func (s *scheduler) exit(t *Task, f Frame, p *Panic) {
	b := f.Base()
	if p != nil {
		if p.Prev == nil && b.panicking != nil && b.panicking != p {
			p.Prev = b.panicking
		}
		b.panicking = p
	}
	t.Frame = f
	if len(b.Defers) > 0 {
		r := &deferRunner{target: f}
		r.parent = f
		t.Frame = r
		return
	}
	s.finish(t, f)
}

// finish pops f and delivers its results, or its panic, to the caller.
func (s *scheduler) finish(t *Task, f Frame) {
	b := f.Base()
	p := b.panicking
	parent := b.parent
	t.Frame = parent
	if parent == nil {
		t.done = true
		if p != nil {
			if s.harness {
				panic(fatalPanicSignal{p})
			}
			reportChain(p)
		}
		return
	}
	if r, ok := parent.(*deferRunner); ok && r.child == f {
		r.childDone(t, p)
		return
	}
	if p != nil {
		s.exit(t, parent, p)
		return
	}
	t.RV = f.Results()
}

func reportChain(p *Panic) {
	var chain []*Panic
	for q := p; q != nil; q = q.Prev {
		chain = append([]*Panic{q}, chain...)
	}
	out := ""
	for i, q := range chain {
		if i > 0 {
			out += "\t"
		}
		out += "panic: " + FormatPanic(q.Value)
		if q.Recovered {
			out += " [recovered]"
		}
		out += "\n"
	}
	os.Stderr.WriteString(out)
	os.Exit(2)
}

// deferRunner runs a frame's deferred calls in reverse order.
type deferRunner struct {
	FrameBase
	target     Frame
	child      Frame
	savedPanic *Panic
	savedFid   int
}

func (r *deferRunner) Results() []any { return nil }

func (r *deferRunner) Step(t *Task) {
	tb := r.target.Base()
	for len(tb.Defers) > 0 {
		d := tb.Defers[len(tb.Defers)-1]
		tb.Defers = tb.Defers[:len(tb.Defers)-1]
		r.savedPanic, r.savedFid = t.curPanic, t.deferTarget
		t.curPanic, t.deferTarget = tb.panicking, d.Fid
		if d.Start != nil {
			child, p := startProtected(d.Start)
			if p == nil {
				r.child = child
				child.Base().parent = r
				t.Frame = child
				return
			}
			t.curPanic, t.deferTarget = r.savedPanic, r.savedFid
			r.after(tb, p)
			continue
		}
		p := callProtected(d.Call)
		t.curPanic, t.deferTarget = r.savedPanic, r.savedFid
		r.after(tb, p)
	}
	t.Frame = r.target
	sched.finish(t, r.target)
}

func (r *deferRunner) childDone(t *Task, p *Panic) {
	t.curPanic, t.deferTarget = r.savedPanic, r.savedFid
	r.child = nil
	t.Frame = r
	r.after(r.target.Base(), p)
}

func (r *deferRunner) after(tb *FrameBase, p *Panic) {
	if p != nil {
		if p.Prev == nil && tb.panicking != nil && tb.panicking != p {
			p.Prev = tb.panicking
		}
		tb.panicking = p
		return
	}
	if tb.panicking != nil && tb.panicking.Recovered {
		tb.panicking = nil
	}
}

func callProtected(f func()) (p *Panic) {
	defer func() {
		if r := recover(); r != nil {
			p = &Panic{Value: r}
		}
	}()
	f()
	return nil
}

func startProtected(f func() Frame) (fr Frame, p *Panic) {
	defer func() {
		if r := recover(); r != nil {
			p = &Panic{Value: r}
		}
	}()
	return f(), nil
}

// RecoverFrame implements recover for deferred calls run by the runtime;
// fid identifies the function calling recover.
func RecoverFrame(fid int) any {
	t := sched.cur
	if t == nil {
		return nil
	}
	p := t.curPanic
	if p == nil || p.Recovered || t.deferTarget != fid {
		return nil
	}
	p.Recovered = true
	return p.Value
}

// Call pushes a callee frame; the caller resumes with the callee's results
// in t.RV.
func Call(t *Task, child Frame) {
	child.Base().parent = t.Frame
	t.Frame = child
}

// Ret returns from frame f.
func Ret(t *Task, f Frame) { sched.exit(t, f, nil) }

// As converts a resumed result to T, mapping nil to the zero value.
func As[T any](v any) T {
	if v == nil {
		var zero T
		return zero
	}
	return v.(T)
}

// syncFrame runs an ordinary function as a frame, for go statements and
// function values that must use the resumable convention.
type syncFrame struct {
	FrameBase
	fn  func() []any
	res []any
}

func (f *syncFrame) Results() []any { return f.res }
func (f *syncFrame) Step(t *Task) {
	f.res = f.fn()
	Ret(t, f)
}

// Sync adapts an ordinary call to the resumable calling convention.
func Sync(fn func() []any) Frame { return &syncFrame{fn: fn} }

// Spawn starts a task running frame f: go f(args).
func Spawn(f Frame) {
	s := sched
	t := &Task{id: s.nextID, Frame: f, deferTarget: -1}
	s.nextID++
	s.ready(t)
}

// RunMain runs the program entry as the first task until it returns.
func RunMain(entry Frame) {
	s := &scheduler{rng: seed(), nextID: 1}
	sched = s
	main := &Task{Frame: entry, deferTarget: -1}
	s.main = main
	s.ready(main)
	for !main.done {
		s.run(s.next())
	}
}

func yieldTask(t *Task) {
	sched.ready(t)
	sched.block(t)
}

// harnessFrame runs one pause primitive for a conformance case.
type harnessFrame struct {
	FrameBase
	fn  func(t *Task)
	res []any
}

func (h *harnessFrame) Results() []any { return h.res }
func (h *harnessFrame) Step(t *Task) {
	if h.PC == 0 {
		h.PC = 1
		h.fn(t)
		return
	}
	h.res = t.RV
	Ret(t, h)
}

// Await runs fn in a fresh isolated scheduler and returns the results the
// primitive delivered. It reports blocked when no task can run, and
// panics with the source panic value when the case panics.
func Await(fn func(t *Task)) (rv []any, blocked bool) {
	s := &scheduler{rng: seed(), nextID: 1, harness: true}
	sched = s
	h := &harnessFrame{fn: fn}
	main := &Task{Frame: h, deferTarget: -1}
	s.main = main
	s.ready(main)
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case deadlockSignal:
				blocked = true
			case fatalPanicSignal:
				panic(x.p.Value)
			default:
				panic(r)
			}
		}
	}()
	for !main.done {
		s.run(s.next())
	}
	return h.res, false
}

// ResetScheduler installs a fresh isolated scheduler for a conformance case
// that uses no pause primitive.
func ResetScheduler() {
	sched = &scheduler{rng: seed(), nextID: 1, harness: true}
	sched.cur = &Task{deferTarget: -1}
}
