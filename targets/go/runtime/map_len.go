package rt

func (m *Map[K, V]) Len() int {
	if m == nil {
		return 0
	}
	return len(m.index)
}
