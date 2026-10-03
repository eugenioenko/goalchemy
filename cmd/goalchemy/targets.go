package main

import (
	"strings"

	"github.com/eugenioenko/goalchemy/internal/diagnostics"
	"github.com/eugenioenko/goalchemy/internal/driver"
)

func targetNames() string { return strings.Join(driver.Targets(), ", ") }

func emitTarget(name string, res *driver.Result, out string, format bool) []diagnostics.Diagnostic {
	ds := driver.Emit(name, res, out)
	if format && !diagnostics.HasErrors(ds) {
		ds = append(ds, driver.Format(name, out)...)
	}
	return ds
}
