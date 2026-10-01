package main

// goalchemy:gate cooperative

import "goalchemy/lib/sync"

func main() {
	var mu sync.Mutex
	defer func() { println("recover cannot catch:", recover() == nil) }()
	mu.Lock()
	mu.Unlock()
	println("unlocking again")
	mu.Unlock()
}
