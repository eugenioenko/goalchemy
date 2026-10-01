package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"goalchemy/internal/catalog"
	"goalchemy/internal/project"
)

func runFeatures(_ context.Context, _ []string, stdout, stderr io.Writer) int {
	f, err := project.LoadFeatures(catalog.FS())
	if err != nil {
		fmt.Fprintln(stderr, "goalchemy features:", err)
		return 1
	}
	fmt.Fprintf(stdout, "Goalchemy %s supported features\n\n", f.Release)
	for _, g := range f.Gates {
		fmt.Fprintf(stdout, "gate %-12s %-9s %s\n", g.Name, g.Status, strings.Join(g.Targets, ", "))
	}
	fmt.Fprintln(stdout)
	for _, ft := range f.Features {
		fmt.Fprintf(stdout, "%-38s %-12s %-10s %s\n", ft.ID, ft.Gate, ft.Status, strings.Join(ft.Targets, ", "))
	}
	return 0
}
