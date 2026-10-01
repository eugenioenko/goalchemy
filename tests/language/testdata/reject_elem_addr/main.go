package main

// goalchemy:reject GCS006

type T struct{ n int }

func (t *T) inc() { t.n++ }

func main() {
	ts := []T{{1}}
	ts[0].inc()
	println(ts[0].n)
}
