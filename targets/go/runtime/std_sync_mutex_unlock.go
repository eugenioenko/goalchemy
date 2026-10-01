package rt

func StdSyncMutexUnlock(m *Mutex) {
	if !m.locked {
		fatal("sync: unlock of unlocked mutex")
	}
	if len(m.waiters) > 0 {
		t := m.waiters[0]
		m.waiters = m.waiters[1:]
		sched.ready(t)
		return
	}
	m.locked = false
}
