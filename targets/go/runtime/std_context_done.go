package rt

func StdContextContextDone(c Context) Chan[struct{}] { return c.c.done }
