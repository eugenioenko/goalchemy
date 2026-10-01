package main

// goalchemy:gate cooperative

func trySend(ch chan int, v int) bool {
	select {
	case ch <- v:
		return true
	default:
		return false
	}
}

func tryRecv(ch chan int) (int, bool) {
	select {
	case v := <-ch:
		return v, true
	default:
		return 0, false
	}
}

func main() {
	ch := make(chan int, 2)
	println(trySend(ch, 1), trySend(ch, 2), trySend(ch, 3))
	v, ok := tryRecv(ch)
	println(v, ok)
	v, ok = tryRecv(ch)
	println(v, ok)
	v, ok = tryRecv(ch)
	println(v, ok)
	var nilc chan int
	select {
	case nilc <- 1:
		println("no")
	default:
		println("nil channel never ready")
	}
}
