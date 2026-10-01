package rt

// Set stores v under k. Updating keeps the entry's iteration position.
func (m Map[K, V]) Set(k K, v V) {
	checkKey(k)
	if m.d == nil {
		panic(PlainError("assignment to entry in nil map"))
	}
	if e, ok := m.d.index[k]; ok {
		e.value = v
		return
	}
	e := &mapEntry[K, V]{key: k, value: v, live: true}
	m.d.index[k] = e
	m.d.entries = append(m.d.entries, e)
}
