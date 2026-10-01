package rt

// Get returns the value for k and whether it is present.
func (m *Map[K, V]) Get(k K) (V, bool) {
	if m == nil {
		var zero V
		_ = map[K]bool{k: true}
		return zero, false
	}
	if e, ok := m.index[k]; ok {
		return e.value, true
	}
	var zero V
	return zero, false
}
