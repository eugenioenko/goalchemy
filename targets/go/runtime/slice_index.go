package rt

func SliceIndex[T any, I Integer](s []T, i I) T { return s[i] }
