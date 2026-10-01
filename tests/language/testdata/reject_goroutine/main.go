package main

// goalchemy:reject GCS011

func main() {
	done := make(chan bool)
	go func() { done <- true }()
	<-done
}
