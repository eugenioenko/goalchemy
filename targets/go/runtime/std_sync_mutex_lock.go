package rt

// Mutex is sync.Mutex: locked state and a FIFO queue of waiting tasks.
// Unlock hands the lock directly to the first waiter.
type Mutex struct {
	locked  bool
	waiters []*Task
}

// StdSyncMutexLock is a pause primitive.
func StdSyncMutexLock(t *Task, m *Mutex) {
	if !m.locked {
		m.locked = true
		return
	}
	m.waiters = append(m.waiters, t)
	sched.block(t)
}
