package main

// goalchemy:reject GCS006

import "github.com/eugenioenko/goalchemy/std/fmt"

type point struct{ x, y int }

func main() {
	println(fmt.Sprintf("%v", point{1, 2}))
}
