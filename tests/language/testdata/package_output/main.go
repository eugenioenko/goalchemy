// goalchemy:gate cooperative
package main

import (
	first "github.com/eugenioenko/goalchemy/tests/language/testdata/package_output/first/value"
	"github.com/eugenioenko/goalchemy/tests/language/testdata/package_output/order"
	second "github.com/eugenioenko/goalchemy/tests/language/testdata/package_output/second/value"
	"github.com/eugenioenko/goalchemy/tests/language/testdata/package_output/worker"
)

var Started = order.Record("/root-var") + worker.Started

func init() { order.Record("/root-init") }

func must(ok bool) {
	if !ok {
		panic("package output")
	}
}

func main() {
	must(order.Log == "order-var/order-init/first-var/first-init/second-var/second-init/worker-var/worker-init/root-var/root-init")
	must(order.Count == 8 && first.Counter == 13 && second.Counter == 15 && Started == 27)
	capture := worker.Capture(3)
	must(capture() == 16 && capture() == 20 && first.Counter == 20)
	must(worker.Suspended()() == 20)
	v := first.Value{Count: 4, Array: [2]int{5, 6}, Slice: []int{7}}
	copy := worker.Copy(v)
	must(v.Count == 4 && v.Array[0] == 5 && copy.Count == 5 && copy.Array[0] == 6)
	must(v.Slice[0] == 8 && copy.Slice[0] == 8)
	bound := v.Amount
	v.Count = 9
	must(bound() == 4 && v.Amount() == 9)
	envelope := worker.Envelope{Value: v}
	var reader first.Reader = &envelope
	must(reader.Amount() == 9 && first.Private(envelope) == 10)
	must(!second.IsValue(v) && second.IsValue(second.Value{}))
	var nilValue *worker.Envelope
	var typedNil first.Reader = nilValue
	must(typedNil != nil)
	m := map[any]int{first.Key{Count: 1}: 11, second.Key{Count: 1}: 22}
	must(len(m) == 2 && m[first.Key{Count: 1}] == 11 && m[second.Key{Count: 1}] == 22)
	println("package output ok")
}
