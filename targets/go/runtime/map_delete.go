package rt

func (m Map[K, V]) Delete(k K) {
	checkKey(k)
	if m.d == nil {
		return
	}
	if e, ok := m.d.index[k]; ok {
		e.live = false
		delete(m.d.index, k)
		m.d.compact()
	}
}
