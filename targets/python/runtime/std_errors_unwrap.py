"""std.errors.unwrap: the result of an Unwrap() error method, or nil."""


def std_errors_unwrap(err):
    if err is None:
        return None
    unwrap = err.t.methods.get("Unwrap")
    return None if unwrap is None else unwrap(err.v)
