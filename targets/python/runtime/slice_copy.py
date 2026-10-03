"""core.slice.copy: retain a temporary for aliased byte backing only."""


def copy(dst, src, clone=None):
    n = min(dst.l, src.l)
    if n == 0:
        return 0
    if dst.b:
        source = src.a[src.o:src.o + n] if dst.a is src.a else memoryview(src.a)[src.o:src.o + n]
        memoryview(dst.a)[dst.o:dst.o + n] = source
        return n
    tmp = src.a[src.o:src.o + n]
    for i in range(n):
        dst.a[dst.o + i] = clone(tmp[i]) if clone else tmp[i]
    return n


def copy_string(dst, s):
    n = min(dst.l, len(s))
    if n == 0:
        return 0
    if dst.b:
        memoryview(dst.a)[dst.o:dst.o + n] = s[:n]
        return n
    for i in range(n):
        dst.a[dst.o + i] = s[i]
    return n
