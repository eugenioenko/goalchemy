package main

type Box[T any] struct{ v T } // want GCS003

func Map[T any](x T) T { return x } // want GCS003

type Number interface{ ~int | ~int64 } // want GCS003

func main() {
	_ = Map(1)     // want GCS003
	var b Box[int] // want GCS003
	_ = b
}
