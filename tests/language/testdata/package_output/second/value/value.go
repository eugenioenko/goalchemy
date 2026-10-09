package value

import (
	first "github.com/eugenioenko/goalchemy/tests/language/testdata/package_output/first/value"
	"github.com/eugenioenko/goalchemy/tests/language/testdata/package_output/order"
)

var Counter = first.Counter + order.Record("/second-var")

func init() {
	order.Record("/second-init")
	Counter++
}

// These have identical underlying representations but distinct Go identities.
type Value struct {
	Count int
	Array [2]int
	Slice []int
}

type Key struct{ Count int }

func IsValue(v any) bool { _, ok := v.(Value); return ok }
