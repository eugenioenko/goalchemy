package rt

func (m *Map[K, V]) Clear() {
	if m == nil {
		return
	}
	for _, e := range m.entries {
		e.live = false
	}
	clear(m.index)
	m.entries = nil
}
