// Package goalchemy embeds the contract catalog and target runtimes so the
// compiler binary is self-contained.
package goalchemy

import "embed"

//go:embed specs targets lib
var Assets embed.FS
