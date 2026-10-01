package rt

// StdTimeSleep is a pause primitive suspending for d nanoseconds of virtual
// time. Virtual time advances only when no task can run.
func StdTimeSleep(t *Task, d int64) {
	if d <= 0 {
		yieldTask(t)
		return
	}
	sched.addTimer(d, t, nil)
	sched.block(t)
}
