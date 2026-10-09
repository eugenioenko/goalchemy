package catalog

import "go/types"

// ErrorsPackage is the standard errors package compiled as source.
const ErrorsPackage = StdModule + "errors"

// IsErrorsAs reports whether obj is errors.As from std/errors or its native
// lib/errors counterpart. Go implements As with reflection, so the compiler
// expands each call for the static type of its target instead.
func IsErrorsAs(obj types.Object) bool {
	fn, ok := obj.(*types.Func)
	if !ok || fn.Name() != "As" || fn.Pkg() == nil {
		return false
	}
	return fn.Pkg().Path() == ErrorsPackage || fn.Pkg().Path() == LibModule+"errors"
}
