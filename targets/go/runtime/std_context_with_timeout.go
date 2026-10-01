package rt

func StdContextWithTimeout(parent Context, d int64) (Context, func()) {
	c := newChild(parent)
	if c.err == nil {
		sched.addTimer(d, nil, func() { c.cancel(ContextDeadlineExceeded) })
	}
	return Context{c}, func() { c.cancel(ContextCanceled) }
}
