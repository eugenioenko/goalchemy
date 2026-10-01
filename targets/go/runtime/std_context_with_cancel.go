package rt

func StdContextWithCancel(parent Context) (Context, func()) {
	c := newChild(parent)
	return Context{c}, func() { c.cancel(ContextCanceled) }
}
