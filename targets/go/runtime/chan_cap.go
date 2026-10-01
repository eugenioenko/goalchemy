package rt

func (c Chan[T]) Cap() int {
	if c.c == nil {
		return 0
	}
	return c.c.size
}
