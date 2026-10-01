package main

// goalchemy:gate cooperative
// goalchemy:golden

import (
	"runtime"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for step := 0; step < 2; step++ {
				println("task", i, "step", step)
				runtime.Gosched()
			}
		}()
	}
	println("main spawned")
	wg.Wait()
	a, b := make(chan int, 10), make(chan int, 10)
	for i := 0; i < 10; i++ {
		a <- i
		b <- i
	}
	picks := ""
	for i := 0; i < 10; i++ {
		select {
		case <-a:
			picks += "a"
		case <-b:
			picks += "b"
		}
	}
	println(picks)
}
