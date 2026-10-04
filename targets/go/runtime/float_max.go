package rt

func FloatMax[T Floating](a, b T) T { return T(max(a, b)) }
