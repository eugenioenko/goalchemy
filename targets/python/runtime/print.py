"""core.print: print and println of Booleans, integers, and strings."""

from ..types.print import write_stderr


def print_string(v):
    if isinstance(v, bytes):
        return v
    if v is True:
        return b"true"
    if v is False:
        return b"false"
    return str(v).encode()


def go_print(args, newline):
    parts = [print_string(a) for a in args]
    out = (b" " if newline else b"").join(parts)
    if newline:
        out += b"\n"
    write_stderr(out)
