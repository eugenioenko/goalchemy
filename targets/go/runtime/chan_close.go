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
	var zero T
	for w := dequeue(&ch.recvq); w != nil; w = dequeue(&ch.recvq) {
		w.recvDone(zero, false)
	}
	for w := dequeue(&ch.sendq); w != nil; w = dequeue(&ch.sendq) {
		w.sendDone(true)
	}
}
