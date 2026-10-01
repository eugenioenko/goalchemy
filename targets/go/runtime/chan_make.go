package rt

// selectState is shared by the waiters a blocked select registers.
type selectState struct {
	done  bool
	index int
}

type waiter struct {
	t      *task
	val    any
	ok     bool
	closed bool
	sel    *selectState
	idx    int
}

type chanCore struct {
	buf    []any
	size   int
	closed bool
	recvq  []*waiter
	sendq  []*waiter
}

// Chan is a channel value: a handle to shared channel state. The zero Chan
// is the nil channel.
type Chan[T any] struct {
	c *chanCore
}

func (c Chan[T]) core() *chanCore { return c.c }

// MakeChan implements make(chan T, size).
func MakeChan[T any](size int) Chan[T] {
	if size < 0 {
		panic(PlainError("makechan: size out of range"))
	}
	return Chan[T]{&chanCore{size: size}}
}

// dequeue removes the first waiter that can still complete.
func dequeue(q *[]*waiter) *waiter {
	for len(*q) > 0 {
		w := (*q)[0]
		*q = (*q)[1:]
		if w.sel == nil || !w.sel.done {
			return w
		}
	}
	return nil
}

func (w *waiter) complete(val any, ok, closed bool) {
	if w.sel != nil {
		w.sel.done = true
		w.sel.index = w.idx
	}
	w.val, w.ok, w.closed = val, ok, closed
	sched.ready(w.t)
}
