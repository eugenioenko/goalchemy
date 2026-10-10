// SPDX-License-Identifier: Apache-2.0

// Package json is the native Go implementation behind std/encoding/json used
// when the program runs with the Go toolchain. The compiler expands calls to
// std/encoding/json.Marshal and MarshalIndent for the static type of their
// operand; these functions are not capabilities.
package json

import (
	"encoding/json"
	"errors"
	"strings"
)

// Error kinds reported by Marshal and MarshalIndent.
const (
	OK = iota
	UnsupportedType
	UnsupportedValue
	MarshalerFailed
	MarshalerSyntax
	Other
)

// Marshal returns the JSON encoding of v. A failure is reported by kind with
// the type name, the marshaler method or value text, the syntax error message
// and offset, and the underlying error, so std/encoding/json can rebuild its
// own error values.
func Marshal(v any) ([]byte, int, string, string, string, int64, error) {
	b, err := json.Marshal(v)
	return classify(b, err)
}

// MarshalIndent is like Marshal but formats the output with Indent.
func MarshalIndent(v any, prefix, indent string) ([]byte, int, string, string, string, int64, error) {
	b, err := json.MarshalIndent(v, prefix, indent)
	return classify(b, err)
}

func classify(b []byte, err error) ([]byte, int, string, string, string, int64, error) {
	if err == nil {
		return b, OK, "", "", "", 0, nil
	}
	var ute *json.UnsupportedTypeError
	if errors.As(err, &ute) && error(ute) == err {
		return nil, UnsupportedType, ute.Type.String(), "", "", 0, nil
	}
	var uve *json.UnsupportedValueError
	if errors.As(err, &uve) && error(uve) == err {
		return nil, UnsupportedValue, "", uve.Str, "", 0, nil
	}
	var me *json.MarshalerError
	if errors.As(err, &me) && error(me) == err {
		src := strings.TrimPrefix(me.Error(), "json: error calling ")
		src = src[:strings.Index(src, " ")]
		var se *json.SyntaxError
		if errors.As(me.Err, &se) && error(se) == me.Err {
			return nil, MarshalerSyntax, me.Type.String(), src, se.Error(), se.Offset, nil
		}
		return nil, MarshalerFailed, me.Type.String(), src, "", 0, me.Err
	}
	return nil, Other, "", "", err.Error(), 0, nil
}
