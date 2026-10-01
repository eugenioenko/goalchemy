package rt

import "errors"

func StdErrorsIs(err, target error) bool { return errors.Is(err, target) }
