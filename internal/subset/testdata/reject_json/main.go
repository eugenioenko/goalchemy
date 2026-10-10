package main

import (
	"github.com/eugenioenko/goalchemy/std/encoding/json"
	"github.com/eugenioenko/goalchemy/std/encoding/jsonvalue"
)

type Fn struct {
	F func()
}

type Ignored struct {
	F func() `json:"-"`
	f func()
}

type Keys struct {
	M map[float64]int
}

type Zero struct {
	N int `json:"n,omitzero"`
}

type BadName struct {
	N int `json:"'quoted'"`
}

type Inline struct {
	N int `json:",inline"`
}

type Twice struct {
	N int `json:",omitempty,omitempty"`
}

type Hidden struct {
	n int `json:"n"`
}

type NumberString struct {
	N json.Number `json:",string"`
}

type Point struct{ X, Y int }

type Text int

func (*Text) MarshalText() ([]byte, error) { return nil, nil }

type Value int

func (Value) MarshalJSON() ([]byte, error) { return nil, nil }

func main() {
	_, _ = json.Marshal(Fn{}) // want GCS006
	_, _ = json.Marshal(Ignored{})
	_, _ = json.Marshal(Keys{})                        // want GCS006
	_, _ = json.Marshal(Zero{})                        // want GCS006
	_, _ = json.Marshal(BadName{})                     // want GCS006
	_, _ = json.Marshal(Inline{})                      // want GCS006
	_, _ = json.Marshal(Twice{})                       // want GCS006
	_, _ = json.Marshal(Hidden{})                      // want GCS006
	_, _ = json.Marshal(NumberString{})                // want GCS006
	_, _ = json.Marshal([]any{Point{}})                // want GCS006
	_, _ = json.Marshal(map[string]any{"p": &Point{}}) // want GCS006
	_, _ = json.Marshal(any(Text(1)))                  // want GCS006
	_, _ = json.Marshal(struct{ A any }{A: Point{}})   // want GCS006
	_, _ = json.Marshal([]any{1, "s", nil, []byte("b"), map[string]any{"v": Value(1)}, jsonvalue.Null(), json.Number("1")})
	_, _ = json.MarshalIndent(Point{}, "", " ")
	f := json.Marshal // want GCS006
	_ = f
	defer json.Marshal(1) // want GCS006
}
