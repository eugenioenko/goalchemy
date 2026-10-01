package rt

func SliceStore[T any, I Integer](s []T, i I, v T) { s[i] = v }
