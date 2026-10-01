package rt

// Iter snapshots the live entries in insertion order.
func (m *Map[K, V]) Iter() *MapIter[K, V] {
	if m == nil {
		return &MapIter[K, V]{}
	}
	return &MapIter[K, V]{entries: append([]*mapEntry[K, V](nil), m.entries...)}
}

// Next returns the next entry still present, reading its current value.
func (it *MapIter[K, V]) Next() (bool, K, V) {
	for it.i < len(it.entries) {
		e := it.entries[it.i]
		it.i++
		if e.live {
			return true, e.key, e.value
		}
	}
	var k K
	var v V
	return false, k, v
}

// MapKeys returns the keys visited by a full iteration of m.
func MapKeys[K comparable, V any](m *Map[K, V]) []K {
	keys := make([]K, 0, m.Len())
	it := m.Iter()
	for {
		ok, k, _ := it.Next()
		if !ok {
			return keys
		}
		keys = append(keys, k)
	}
}
