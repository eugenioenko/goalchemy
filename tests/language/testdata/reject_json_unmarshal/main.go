package main

// goalchemy:reject GCS006

import "github.com/eugenioenko/goalchemy/std/encoding/json"

type Point struct{ X, Y int }

func main() {
	var p Point
	println(json.Unmarshal([]byte(`{"X":1}`), p) == nil)
}
