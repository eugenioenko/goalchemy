// Package goalchemy embeds the contract catalog and target runtimes so the
// compiler binary is self-contained.
package goalchemy

import "embed"

//go:embed LICENSE specs targets lib std tests/language/language_test.go tests/language/testdata/floats tests/language/testdata/co_floats tests/language/testdata/floats_panic32 tests/language/testdata/floats_panic64 tests/language/testdata/floats_panic_named32 tests/language/testdata/floats_panic_named64 tests/integration/float_library_test.go tests/integration/testdata/float_library
var Assets embed.FS
