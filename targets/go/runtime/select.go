package rt

// SelectCase describes one communication; Ch is the channel handle's core.
type SelectCase struct {
	Ch   *chanCore
	Send bool
	Val  any
	Zero any
}

// Case adapts a typed channel send for Select.
func Case[T any](c Chan[T], send bool, v T) SelectCase {
	var zero T
	return SelectCase{Ch: c.c, Send: send, Val: v, Zero: zero}
}

// RecvCase adapts a typed channel receive for Select.
func RecvCase[T any](c Chan[T]) SelectCase {
	var zero T
	return SelectCase{Ch: c.c, Zero: zero}
}

// Select commits one ready case, chosen uniformly by the scheduler's choice
// source; the results {index, value, ok} arrive in t.RV, with index -1 for
// the default. It is a pause primitive.
func Select(t *Task, cases []SelectCase, hasDefault bool) {
	s := sched
	var ready []int
	for i, c := range cases {
		ch := c.Ch
		if ch == nil {
			continue
		}
		if c.Send {
			if ch.closed || hasLive(ch.recvq) || len(ch.buf) < ch.size {
				ready = append(ready, i)
			}
		} else if len(ch.buf) > 0 || hasLive(ch.sendq) || ch.closed {
			ready = append(ready, i)
		}
	}
	if len(ready) > 0 {
		i := ready[s.choose(len(ready))]
		c := cases[i]
		if c.Send {
			if c.Ch.closed {
				panic(PlainError("send on closed channel"))
			}
			if w := dequeue(&c.Ch.recvq); w != nil {
				w.recvDone(c.Val, true)
			} else {
				c.Ch.buf = append(c.Ch.buf, c.Val)
			}
			t.RV = []any{i, nil, false}
			return
		}
		v, ok, _ := c.Ch.tryRecv()
		if !ok {
			v = c.Zero
		}
		t.RV = []any{i, v, ok}
		return
	}
	if hasDefault {
		t.RV = []any{-1, nil, false}
		return
	}
	st := &selectState{}
	for i, c := range cases {
		if c.Ch == nil {
			continue
		}
		w := &waiter{t: t, sel: st, idx: i}
		if c.Send {
			w.val = c.Val
			c.Ch.sendq = append(c.Ch.sendq, w)
		} else {
			c.Ch.recvq = append(c.Ch.recvq, w)
		}
	}
	s.block(t)
	t.cleanup = func() {
		for _, c := range cases {
			if c.Ch != nil {
				unregister(&c.Ch.sendq, st)
				unregister(&c.Ch.recvq, st)
			}
		}
	}
}

func hasLive(q []*waiter) bool {
	for _, w := range q {
		if w.sel == nil || !w.sel.done {
			return true
		}
	}
	return false
}

// unregister removes a select's losing registrations.
func unregister(q *[]*waiter, st *selectState) {
	out := (*q)[:0]
	for _, x := range *q {
		if x.sel != st {
			out = append(out, x)
		}
	}
	*q = out
}
