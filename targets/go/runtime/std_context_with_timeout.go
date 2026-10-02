package rt

func StdContextWithTimeout(parent Context, d int64) (Context, func()) {
	c := newChild(parent)
	if c.err == nil {
		now := sched.now()
		deadline := now + d
		if d > 0 && deadline < now {
			deadline = int64(^uint64(0) >> 1)
		}
		if !c.hasDeadline || deadline < c.deadline {
			c.deadline, c.hasDeadline = deadline, true
		}
		if c.deadline <= now {
			c.cancel(ContextDeadlineExceeded)
		} else {
			c.timer = sched.addTimer(c.deadline-now, nil, func() { c.cancel(ContextDeadlineExceeded) })
			// addTimer observes now again in host mode; preserve the exact
			// creation deadline instead of moving it by setup work.
			c.timer.at = c.deadline
		}
	}
	return Context{c}, func() { c.cancel(ContextCanceled) }
}
