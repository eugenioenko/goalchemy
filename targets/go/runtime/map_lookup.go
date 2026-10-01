package rt

// Get returns the value for k and whether it is present.
func (m Map[K, V]) Get(k K) (V, bool) {
	checkKey(k)
	if m.d == nil {
		var zero V
		return zero, false
	}
	if e, ok := m.d.index[k]; ok {
		return e.value, true
	}
	var zero V
	return zero, false
}
