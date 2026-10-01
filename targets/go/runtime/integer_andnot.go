package rt

func IntAndNot[T Integer](a, b T) T { return a &^ b }
