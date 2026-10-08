package api

import (
	"github.com/eugenioenko/goalchemy/tests/integration/testdata/package_library/order"
	"github.com/eugenioenko/goalchemy/tests/integration/testdata/package_library/state"
)

var Started = order.Record("/api-var")

func init() { order.Record("/api-init") }

type Value struct {
	Count int
	Bytes []byte
	Order string
}
type Failure struct {
	Code  string
	Bytes []byte
}

func (e *Failure) Error() string { return e.Code }

var failure = Failure{Code: "saved", Bytes: []byte{9}}

func Export(v Value) (Value, error) {
	v.Count = state.Run(v.Count)
	v.Order = order.Log
	v.Bytes = state.Fixed[:]
	state.Fixed[0] = byte(Started)
	return v, nil
}

func ErrorGlobal() ([]byte, error) { return nil, &failure }
