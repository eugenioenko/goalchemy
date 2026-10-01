package main

import (
	"strings"

	"github.com/eugenioenko/goalchemy/internal/diagnostics"
	"github.com/eugenioenko/goalchemy/internal/driver"
)

func targetNames() string { return strings.Join(driver.Targets(), ", ") }

func emitTarget(name string, res *driver.Result, out string) []diagnostics.Diagnostic {
	return driver.Emit(name, res, out)
}
