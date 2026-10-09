package worker

import (
	first "github.com/eugenioenko/goalchemy/tests/language/testdata/package_output/first/value"
	"github.com/eugenioenko/goalchemy/tests/language/testdata/package_output/order"
	second "github.com/eugenioenko/goalchemy/tests/language/testdata/package_output/second/value"
)

var Started = order.Record("/worker-var") + second.Counter

func init() {
	order.Record("/worker-init")
	first.Counter += 2
}

type Envelope struct{ first.Value }

func Capture(delta int) func() int {
	p := &first.Counter
	return func() int {
		*p += delta
		delta++
		return *p
	}
}

func Suspended() func() int {
	return func() int {
		q := make(chan int, 1)
		q <- first.Counter
		return <-q
	}
}

func Copy(v first.Value) first.Value {
	v.Count++
	v.Array[0]++
	v.Slice[0]++
	return v
}
