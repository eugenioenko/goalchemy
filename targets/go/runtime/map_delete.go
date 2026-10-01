package rt

func (m *Map[K, V]) Delete(k K) {
	if m == nil {
		_ = map[K]bool{k: true}
		return
	}
	if e, ok := m.index[k]; ok {
		e.live = false
		delete(m.index, k)
		m.compact()
	}
}
