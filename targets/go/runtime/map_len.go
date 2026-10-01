package rt

func (m Map[K, V]) Len() int {
	if m.d == nil {
		return 0
	}
	return len(m.d.index)
}
