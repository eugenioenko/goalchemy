package rt

func SliceSlice[T any](s []T, lo, hi, max int) []T { return s[lo:hi:max] }
