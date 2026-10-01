package main

type T struct{ n int }

func (t *T) Inc() { t.n++ }

func main() {
	xs := []T{{1}}
	p := &xs[0]    // want GCS006
	q := &xs[0].n  // want GCS006
	xs[0].Inc()    // want GCS006
	arr := [2][3]int{}
	s := arr[0][:] // want GCS006
	ps := []*T{{1}}
	ps[0].Inc()
	r := &ps[0].n
	_, _, _, _ = p, q, s, r
}
