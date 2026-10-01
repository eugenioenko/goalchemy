package rt

func SliceCopy[T any](dst, src []T) int { return copy(dst, src) }
