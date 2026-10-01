package main

// goalchemy:gate cooperative

import "goalchemy/lib/task"

func main() {
	total := 0
	task.All(
		func() { total += 1 },
		func() { total += 10 },
		func() { total += 100 },
	)
	println("total", total)
	results := make([]int, 4)
	var jobs []func()
	for i := range results {
		jobs = append(jobs, func() { results[i] = i * i })
	}
	task.All(jobs...)
	println(results[0], results[1], results[2], results[3])
	ch := make(chan int)
	sum := 0
	task.All(
		func() {
			for i := 1; i <= 3; i++ {
				ch <- i
			}
			close(ch)
		},
		func() {
			for v := range ch {
				sum += v
			}
		},
	)
	println("sum", sum)
	task.All()
	println("done")
}
