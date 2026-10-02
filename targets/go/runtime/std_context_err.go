package rt

import "context"

// Context is a cancellation context. Done returns a channel closed on
// cancellation; Background's channel is nil.
type Context struct{ c *ctxState }

type ctxState struct {
	done        Chan[struct{}]
	err         error
	children    map[*ctxState]struct{}
	parent      *ctxState
	owner       *scheduler
	deadline    int64
	hasDeadline bool
	timer       *timer
	hooks       map[uint64]func()
}

var (
	ContextCanceled         = context.Canceled
	ContextDeadlineExceeded = context.DeadlineExceeded
	background              = &ctxState{}
)

func (s *ctxState) cancel(err error) {
	if s.err != nil {
		return
	}
	s.err = err
	if s.parent != nil {
		delete(s.parent.children, s)
		s.parent = nil
	}
	if s.timer != nil {
		s.owner.removeTimer(s.timer)
		s.timer = nil
	}
	s.done.Close()
	for _, cancel := range s.hooks {
		cancel()
	}
	s.hooks = nil
	for c := range s.children {
		c.cancel(err)
	}
	s.children = nil
}

func newChild(parent Context) *ctxState {
	c := &ctxState{done: MakeChan[struct{}](0), owner: sched}
	p := parent.c
	if p == nil {
		panic(PlainError("cannot create context from nil parent"))
	}
	if p != background && p.owner != sched {
		panic(HostFault{Value: "context belongs to another scheduler"})
	}
	c.deadline, c.hasDeadline = p.deadline, p.hasDeadline
	if p.err != nil {
		c.cancel(p.err)
	} else if p.hasDeadline && sched.now() >= p.deadline {
		p.cancel(ContextDeadlineExceeded)
		c.cancel(ContextDeadlineExceeded)
	} else if p != background {
		if p.children == nil {
			p.children = make(map[*ctxState]struct{})
		}
		p.children[c] = struct{}{}
		c.parent = p
	}
	return c
}

func StdContextContextErr(c Context) error {
	if c.c == nil {
		return nil
	}
	if c.c.err == nil && c.c.hasDeadline && c.c.owner.now() >= c.c.deadline {
		c.c.cancel(ContextDeadlineExceeded)
	}
	return c.c.err
}
