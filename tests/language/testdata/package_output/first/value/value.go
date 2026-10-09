package value

import "github.com/eugenioenko/goalchemy/tests/language/testdata/package_output/order"

var Counter = order.Record("/first-var")

func init() {
	order.Record("/first-init")
	Counter += 10
}

type Value struct {
	Count int
	Array [2]int
	Slice []int
}

type Key struct{ Count int }
type Reader interface{ Amount() int }

func (v Value) Amount() int { return v.Count }
func (v Value) secret() int { return v.Count + 1 }

func Private(v interface{ secret() int }) int { return v.secret() }
