package rt

// Recv implements v, ok := <-c.
func (c Chan[T]) Recv() (T, bool) {
	ch := c.c
	s := sched
	var zero T
	if ch == nil {
		s.park()
		return zero, false
	}
	if v, ok, done := ch.tryRecv(); done {
		if !ok {
			return zero, false
		}
		return v.(T), true
	}
	w := &waiter{t: s.cur}
	ch.recvq = append(ch.recvq, w)
	s.park()
	if !w.ok {
		return zero, false
	}
	return w.val.(T), true
}

// tryRecv receives without blocking when a value or closure is available.
func (ch *chanCore) tryRecv() (any, bool, bool) {
	if len(ch.buf) > 0 {
		v := ch.buf[0]
		ch.buf = ch.buf[1:]
		if w := dequeue(&ch.sendq); w != nil {
			ch.buf = append(ch.buf, w.val)
			w.complete(nil, true, false)
		}
		return v, true, true
	}
	if w := dequeue(&ch.sendq); w != nil {
		v := w.val
		w.complete(nil, true, false)
		return v, true, true
	}
	if ch.closed {
		return nil, false, true
	}
	return nil, false, false
}
