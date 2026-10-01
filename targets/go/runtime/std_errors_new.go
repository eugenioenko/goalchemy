package rt

import "errors"

func StdErrorsNew(text string) error { return errors.New(text) }
