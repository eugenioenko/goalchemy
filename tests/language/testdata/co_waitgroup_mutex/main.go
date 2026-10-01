package main

// goalchemy:gate cooperative

import "sync"

type Counter struct {
	mu sync.Mutex
	n  map[string]int
}

func (c *Counter) Inc(k string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n[k]++
}

func main() {
	c := &Counter{n: map[string]int{}}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Inc("total")
				if j%10 == 0 {
					c.Inc("tens")
				}
			}
		}()
	}
	wg.Wait()
	println(c.n["total"], c.n["tens"])
	results := make([]int, 5)
	var wg2 sync.WaitGroup
	for i := range results {
		wg2.Add(1)
		go func(idx int) {
			defer wg2.Done()
			results[idx] = idx * idx
		}(i)
	}
	wg2.Wait()
	println(results[0], results[1], results[2], results[3], results[4])
}
