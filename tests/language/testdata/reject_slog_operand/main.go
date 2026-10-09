package main

// goalchemy:reject GCS006

import "github.com/eugenioenko/goalchemy/std/log/slog"

type point struct{ x, y int }

func main() {
	slog.Warn("moved", "to", point{1, 2})
}
