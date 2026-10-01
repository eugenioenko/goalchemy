package rt

// Send implements c <- v.
func (c Chan[T]) Send(v T) {
	ch := c.c
	s := sched
	if ch == nil {
		s.park()
		return
	}
	if ch.closed {
		panic(PlainError("send on closed channel"))
	}
	if w := dequeue(&ch.recvq); w != nil {
		w.complete(v, true, false)
		return
	}
	if len(ch.buf) < ch.size {
		ch.buf = append(ch.buf, v)
		return
	}
	w := &waiter{t: s.cur, val: v}
	ch.sendq = append(ch.sendq, w)
	s.park()
	if w.closed {
		panic(PlainError("send on closed channel"))
	}
}
