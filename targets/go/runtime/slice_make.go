package rt

func SliceMake[T any, I Integer](n, c I) []T { return make([]T, n, c) }
