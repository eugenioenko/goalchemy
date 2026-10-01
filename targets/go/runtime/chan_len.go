package rt

func (c Chan[T]) Len() int {
	if c.c == nil {
		return 0
	}
	return len(c.c.buf)
}

// ChanBuffered returns a copy of the buffered values, for inspection.
func ChanBuffered[T any](c Chan[T]) []T {
	var out []T
	if c.c != nil {
		for _, v := range c.c.buf {
			out = append(out, v.(T))
		}
	}
	return out
}

// ChanClosed reports whether the channel is closed, for inspection.
func ChanClosed[T any](c Chan[T]) bool { return c.c != nil && c.c.closed }

// ChanWith builds a channel holding buffered values, for inspection tools.
func ChanWith[T any](size int, vals []T, closed bool) Chan[T] {
	c := MakeChan[T](size)
	for _, v := range vals {
		c.c.buf = append(c.c.buf, v)
	}
	c.c.closed = closed
	return c
}
