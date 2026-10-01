package rt

// StdSyncWaitgroupWait is a pause primitive.
func StdSyncWaitgroupWait(t *Task, wg *WaitGroup) {
	if wg.n == 0 {
		return
	}
	wg.waiters = append(wg.waiters, t)
	sched.block(t)
}
