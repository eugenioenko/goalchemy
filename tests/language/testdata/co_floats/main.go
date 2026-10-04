package main

// goalchemy:gate cooperative

type Message struct {
	Values [2]float32
	Wide   float64
}

func main() {
	ch := make(chan float32)
	go func() { ch <- 1.25 }()
	received := <-ch
	println(received)
	close(ch)
	zero, ok := <-ch
	var positive float32
	println(!ok, zero == positive, 1/zero > 0)
	messages := make(chan Message, 1)
	value := Message{Values: [2]float32{1.5, 2.5}, Wide: 3.75}
	messages <- value
	value.Values[0] = 9
	got := <-messages
	println(got.Values[0], got.Values[1], got.Wide, value.Values[0])
	wide := make(chan float64)
	go func() { wide <- got.Wide + 0.5 }()
	select {
	case v := <-wide:
		println(v)
	}
}
