"""core.slice.copy: copy(dst, src) through a temporary for overlap safety."""


def copy(dst, src, clone=None):
    n = min(dst.l, src.l)
    if n == 0:
        return 0
    tmp = src.a[src.o:src.o + n]
    for i in range(n):
        dst.a[dst.o + i] = clone(tmp[i]) if clone else tmp[i]
    return n


def copy_string(dst, s):
    n = min(dst.l, len(s))
    for i in range(n):
        dst.a[dst.o + i] = s[i]
    return n
