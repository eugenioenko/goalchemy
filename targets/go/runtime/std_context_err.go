package rt

import "errors"

// Context is a cancellation context. Done returns a channel closed on
// cancellation; Background's channel is nil.
type Context struct{ c *ctxState }

type ctxState struct {
	done     Chan[struct{}]
	err      error
	children []*ctxState
}

var (
	ContextCanceled         = errors.New("context canceled")
	ContextDeadlineExceeded = errors.New("context deadline exceeded")
	background              = &ctxState{}
)

func (s *ctxState) cancel(err error) {
	if s.err != nil {
		return
	}
	s.err = err
	s.done.Close()
	for _, c := range s.children {
		c.cancel(err)
	}
	s.children = nil
}

func newChild(parent Context) *ctxState {
	c := &ctxState{done: MakeChan[struct{}](0)}
	p := parent.c
	if p.err != nil {
		c.cancel(p.err)
	} else if p != background {
		p.children = append(p.children, c)
	}
	return c
}

func StdContextContextErr(c Context) error { return c.c.err }
