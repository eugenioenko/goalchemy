package rt

func StdContextContextDone(c Context) Chan[struct{}] {
	if c.c == nil {
		return Chan[struct{}]{}
	}
	StdContextContextErr(c)
	return c.c.done
}
