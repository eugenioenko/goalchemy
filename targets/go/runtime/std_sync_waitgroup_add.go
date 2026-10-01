package rt

// WaitGroup is sync.WaitGroup: a counter and the tasks waiting for zero.
type WaitGroup struct {
	n       int64
	waiters []*task
}

func StdSyncWaitgroupAdd(wg *WaitGroup, d int) {
	wg.n += int64(d)
	if wg.n < 0 {
		panic("sync: negative WaitGroup counter")
	}
	if wg.n == 0 {
		for _, t := range wg.waiters {
			sched.ready(t)
		}
		wg.waiters = nil
	}
}
