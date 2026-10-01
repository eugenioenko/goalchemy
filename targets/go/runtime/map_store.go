package rt

// Set stores v under k. Updating keeps the entry's iteration position.
func (m *Map[K, V]) Set(k K, v V) {
	if m == nil {
		panic(PlainError("assignment to entry in nil map"))
	}
	if e, ok := m.index[k]; ok {
		e.value = v
		return
	}
	e := &mapEntry[K, V]{key: k, value: v, live: true}
	m.index[k] = e
	m.entries = append(m.entries, e)
}
