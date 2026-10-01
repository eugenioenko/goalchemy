package main

type Code int

func main() {
	defer println("deferred before custom panic")
	panic(Code(42))
}
