package rt

func StdSyncWaitgroupWait(wg *WaitGroup) {
	if wg.n == 0 {
		return
	}
	wg.waiters = append(wg.waiters, sched.cur)
	sched.park()
}
