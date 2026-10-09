// SPDX-License-Identifier: Apache-2.0

// Package errors provides Goalchemy's error values. This file is the native
// Go implementation used when the program runs with the Go toolchain.
package errors

import "errors"

// New returns a distinct error whose message is text.
func New(text string) error { return errors.New(text) }

// Is reports whether any error in err's Unwrap chain matches target.
func Is(err, target error) bool { return errors.Is(err, target) }

// Unwrap returns the result of err's Unwrap method, or nil.
func Unwrap(err error) error { return errors.Unwrap(err) }

// As is the native implementation behind std/errors.As. The compiler expands
// calls to either for the static type of target; it is not a capability.
func As(err error, target any) bool { return errors.As(err, target) }
