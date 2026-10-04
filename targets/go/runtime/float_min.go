package rt

func FloatMin[T Floating](a, b T) T { return T(min(a, b)) }
