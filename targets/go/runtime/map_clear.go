package rt

func (m Map[K, V]) Clear() {
	if m.d == nil {
		return
	}
	for _, e := range m.d.entries {
		e.live = false
	}
	clear(m.d.index)
	m.d.entries = nil
}
