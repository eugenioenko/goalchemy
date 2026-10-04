// Package goalchemy embeds the contract catalog and target runtimes so the
// compiler binary is self-contained.
package goalchemy

import "embed"

//go:embed specs targets lib std tests/language/language_test.go tests/language/testdata/floats tests/language/testdata/co_floats tests/integration/float_library_test.go tests/integration/testdata/float_library
var Assets embed.FS
