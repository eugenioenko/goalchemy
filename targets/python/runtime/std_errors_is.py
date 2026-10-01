"""std.errors.is: errors.Is over the Unwrap() error chain."""


def std_errors_is(err, target):
    if err is None or target is None:
        return err is target
    comparable = target.t.comparable
    cur = err
    while cur is not None:
        if comparable and cur.t is target.t and cur.t.eq(cur.v, target.v):
            return True
        is_m = cur.t.methods.get("Is")
        if is_m is not None and is_m(cur.v, target):
            return True
        unwrap = cur.t.methods.get("Unwrap")
        if unwrap is None:
            return False
        cur = unwrap(cur.v)
    return False
