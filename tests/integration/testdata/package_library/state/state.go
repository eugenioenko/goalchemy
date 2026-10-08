package state

import "github.com/eugenioenko/goalchemy/tests/integration/testdata/package_library/order"

var Count = order.Record("/state-var")
var Fixed = [3]byte{0, 255, 128}

func init() { order.Record("/state-init"); Count += 10 }

type Value struct {
	Count int
	Bytes []byte
}
type Wrapped struct{ Value }

func (v Value) Amount() int { return v.Count }

func Run(input int) int {
	step := input
	f := func() int { Count += step; step++; return Count }
	first := f()
	var reader interface{ Amount() int } = Wrapped{Value{Count: f()}}
	return first + reader.Amount()
}
