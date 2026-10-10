// Command protoc-gen-goalchemy is a protoc and buf plugin that generates
// Goalchemy-subset Go messages with proto3 JSON encoding and Connect clients
// for unary methods.
//
//	protoc --goalchemy_out=. --goalchemy_opt=go_import_prefix=example.com/app/gen *.proto
//
// Parameters: go_import_prefix=<path> places each file at <path>/<proto
// directory>; include=<full.Name>[;<full.Name>...] limits output to those
// messages, enums, services and methods and everything they reference;
// getters=false omits GetX accessors.
package main

import (
	"google.golang.org/protobuf/compiler/protogen"

	"github.com/eugenioenko/goalchemy/internal/protoc"
)

func main() {
	var opts protoc.Options
	protogen.Options{ParamFunc: opts.Set}.Run(func(plugin *protogen.Plugin) error {
		return protoc.Run(plugin, opts)
	})
}
