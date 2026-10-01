"""std.errors.new: errors.New returns a distinct *errors.errorString."""

from ..types.iface import box, type_desc


class _ErrorString:
    __slots__ = ("s",)

    def __init__(self, s):
        self.s = s


ERRORS_ERROR_STRING = type_desc(
    name="*errors.errorString", kind="pointer", eq=lambda a, b: a is b, key=lambda a: id(a), methods={"Error": lambda p: p.s}
)


def std_errors_new(text):
    return box(ERRORS_ERROR_STRING, _ErrorString(text))
