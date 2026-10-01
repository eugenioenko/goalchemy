package main

// goalchemy:gate cooperative

func churn() int {
	n := 0
	for i := 0; i < 50000; i++ {
		x := make([]int, 2)
		n += len(x)
	}
	return n
}

type Msg struct {
	id   int
	data []int
}

func main() {
	ch := make(chan *Msg, 4)
	done := make(chan int)
	go func() {
		total := 0
		for m := range ch {
			churn()
			for _, v := range m.data {
				total += v
			}
			total += m.id
		}
		done <- total
	}()
	for i := 0; i < 20; i++ {
		m := &Msg{id: i, data: make([]int, 50)}
		for j := range m.data {
			m.data[j] = i * j
		}
		ch <- m
		churn()
	}
	close(ch)
	println("total", <-done)
}
