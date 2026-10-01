package rt

import (
	"os"
	"strconv"
)

// The cooperative scheduler. Each source task runs on its own goroutine,
// but a baton guarantees that exactly one task executes at a time: a task
// runs until it blocks, yields, returns, or panics, and runnable tasks are
// dispatched in FIFO order. The scheduler state is only touched by the task
// holding the baton.

type task struct {
	id   int
	wake chan struct{}
	done bool
}

type timer struct {
	at  int64
	seq int64
	t   *task
	fn  func()
}

type scheduler struct {
	// onDeadlock, when set, replaces the fatal deadlock report.
	onDeadlock func()
	runq       []*task
	cur        *task
	nextID     int
	rng        uint32
	clock      int64
	timers     []*timer
	seq        int64
}

var sched *scheduler

func init() { mainHook = runScheduler }

func seed() uint32 {
	if s := os.Getenv("GOALCHEMY_SEED"); s != "" {
		if n, err := strconv.ParseUint(s, 10, 32); err == nil && n != 0 {
			return uint32(n)
		}
	}
	return 1
}

func runScheduler(entry func()) {
	sched = &scheduler{rng: seed()}
	main := &task{id: 0, wake: make(chan struct{}, 1)}
	sched.cur = main
	sched.nextID = 1
	entry()
}

// ResetScheduler installs a fresh scheduler whose current task is the
// caller; onDeadlock replaces the fatal deadlock report. Harnesses use it to
// run each case in isolation.
func ResetScheduler(onDeadlock func()) {
	sched = &scheduler{rng: seed(), onDeadlock: onDeadlock, nextID: 1}
	sched.cur = &task{wake: make(chan struct{}, 1)}
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

func (s *scheduler) ready(t *task) { s.runq = append(s.runq, t) }

// park suspends the current task until another task readies it.
func (s *scheduler) park() {
	me := s.cur
	s.switchTo(s.next())
	<-me.wake
	s.cur = me
}

// next returns the next runnable task, advancing the virtual clock when
// only timers can make progress.
func (s *scheduler) next() *task {
	for len(s.runq) == 0 {
		if len(s.timers) == 0 {
			if s.onDeadlock != nil {
				s.onDeadlock()
			}
			fatal("all goroutines are asleep - deadlock!")
		}
		s.fireTimers()
	}
	t := s.runq[0]
	s.runq = s.runq[1:]
	return t
}

func (s *scheduler) switchTo(t *task) {
	t.wake <- struct{}{}
}

func (s *scheduler) fireTimers() {
	earliest := s.timers[0]
	for _, t := range s.timers {
		if t.at < earliest.at || t.at == earliest.at && t.seq < earliest.seq {
			earliest = t
		}
	}
	s.clock = earliest.at
	var keep []*timer
	var due []*timer
	for _, t := range s.timers {
		if t.at == s.clock {
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

func (s *scheduler) addTimer(d int64, t *task, fn func()) {
	s.seq++
	s.timers = append(s.timers, &timer{at: s.clock + d, seq: s.seq, t: t, fn: fn})
}

func fatal(msg string) {
	os.Stderr.WriteString("fatal error: " + msg + "\n")
	os.Exit(2)
}

// Go starts a task running f. The parent continues; the child is queued.
func Go(f func()) {
	s := sched
	t := &task{id: s.nextID, wake: make(chan struct{}, 1)}
	s.nextID++
	go func() {
		<-t.wake
		s.cur = t
		defer func() {
			if r := recover(); r != nil {
				reportPanic(r)
			}
			t.done = true
			s.switchTo(s.next())
		}()
		f()
	}()
	s.ready(t)
}

// Gosched yields to every other runnable task.
func Gosched() {
	s := sched
	s.ready(s.cur)
	s.park()
}
