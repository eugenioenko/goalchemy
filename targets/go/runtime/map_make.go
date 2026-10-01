package rt

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{index: map[K]*mapEntry[K, V]{}}
}
