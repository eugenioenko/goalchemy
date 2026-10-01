package rt

import "errors"

func StdErrorsUnwrap(err error) error { return errors.Unwrap(err) }
