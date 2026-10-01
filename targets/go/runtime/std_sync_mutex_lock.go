package rt

// Mutex is sync.Mutex: locked state and a FIFO queue of waiting tasks.
// Unlock hands the lock directly to the first waiter.
type Mutex struct {
	locked  bool
	waiters []*task
}

func StdSyncMutexLock(m *Mutex) {
	if !m.locked {
		m.locked = true
		return
	}
	m.waiters = append(m.waiters, sched.cur)
	sched.park()
}
