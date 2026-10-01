package rt

// StdTimeSleep suspends the current task for d nanoseconds of virtual
// time. Virtual time advances only when no task can run.
func StdTimeSleep(d int64) {
	s := sched
	if d <= 0 {
		Gosched()
		return
	}
	s.addTimer(d, s.cur, nil)
	s.park()
}
