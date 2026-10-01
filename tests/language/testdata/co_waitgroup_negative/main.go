package main

// goalchemy:gate cooperative

import "sync"

func main() {
	var wg sync.WaitGroup
	defer func() { println("recovered:", recover().(string)) }()
	wg.Add(1)
	wg.Done()
	wg.Done()
}
