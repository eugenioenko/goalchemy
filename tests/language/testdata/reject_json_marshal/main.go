package main

// goalchemy:reject GCS006

import "github.com/eugenioenko/goalchemy/std/encoding/json"

type Point struct{ X, Y int }

func main() {
	b, err := json.Marshal([]any{Point{1, 2}})
	println(string(b), err == nil)
}
