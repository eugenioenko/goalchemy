package rt

// StdTimeSleep suspends for d nanoseconds of the selected scheduler clock.
// Default virtual time advances only when no task can run; the explicit host
// entry uses monotonic real elapsed time.
func StdTimeSleep(t *Task, d int64) {
	if d <= 0 {
		yieldTask(t)
		return
	}
	sched.addTimer(d, t, nil)
	sched.block(t)
}
