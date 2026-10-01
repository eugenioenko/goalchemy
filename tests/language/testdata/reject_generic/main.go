package main

// goalchemy:reject GCS003

func Max[T int | string](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func main() { println(Max(1, 2)) }
