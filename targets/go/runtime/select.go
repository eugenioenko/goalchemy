package rt

// SelectCase describes one communication; Ch is the channel handle's core.
type SelectCase struct {
	Ch   *chanCore
	Send bool
	Val  any
}

// Case adapts a typed channel for Select.
func Case[T any](c Chan[T], send bool, v T) SelectCase {
	return SelectCase{Ch: c.c, Send: send, Val: v}
}

// RecvCase adapts a typed channel receive for Select.
func RecvCase[T any](c Chan[T]) SelectCase { return SelectCase{Ch: c.c} }

// Select commits one ready case, chosen uniformly by the scheduler's choice
// source; it returns -1 when hasDefault is set and nothing is ready, and
// suspends otherwise. The received value and ok are returned for receives.
func Select(cases []SelectCase, hasDefault bool) (int, any, bool) {
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
				w.complete(c.Val, true, false)
			} else {
				c.Ch.buf = append(c.Ch.buf, c.Val)
			}
			return i, nil, false
		}
		v, ok, _ := c.Ch.tryRecv()
		return i, v, ok
	}
	if hasDefault {
		return -1, nil, false
	}
	st := &selectState{}
	ws := make([]*waiter, len(cases))
	for i, c := range cases {
		if c.Ch == nil {
			continue
		}
		w := &waiter{t: s.cur, sel: st, idx: i}
		ws[i] = w
		if c.Send {
			w.val = c.Val
			c.Ch.sendq = append(c.Ch.sendq, w)
		} else {
			c.Ch.recvq = append(c.Ch.recvq, w)
		}
	}
	s.park()
	for i, c := range cases {
		if c.Ch != nil {
			unregister(&c.Ch.sendq, ws[i])
			unregister(&c.Ch.recvq, ws[i])
		}
	}
	w := ws[st.index]
	if cases[st.index].Send && w.closed {
		panic(PlainError("send on closed channel"))
	}
	return st.index, w.val, w.ok
}

func hasLive(q []*waiter) bool {
	for _, w := range q {
		if w.sel == nil || !w.sel.done {
			return true
		}
	}
	return false
}

func unregister(q *[]*waiter, w *waiter) {
	out := (*q)[:0]
	for _, x := range *q {
		if x != w {
			out = append(out, x)
		}
	}
	*q = out
}

// Value converts a received select value to T, using the zero value for
// receives that reported closure.
func Value[T any](v any) T {
	if v == nil {
		var zero T
		return zero
	}
	return v.(T)
}
