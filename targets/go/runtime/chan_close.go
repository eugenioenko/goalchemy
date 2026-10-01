package rt

// Close implements close(c).
func (c Chan[T]) Close() {
	ch := c.c
	if ch == nil {
		panic(PlainError("close of nil channel"))
	}
	if ch.closed {
		panic(PlainError("close of closed channel"))
	}
	ch.closed = true
	for {
		w := dequeue(&ch.recvq)
		if w == nil {
			break
		}
		w.complete(nil, false, false)
	}
	for {
		w := dequeue(&ch.sendq)
		if w == nil {
			break
		}
		w.complete(nil, false, true)
	}
}
