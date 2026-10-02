"""core.slice.clear: zero the elements below the length."""


def clear_slice(s, zero):
    if s.l == 0:
        return
    if s.b:
        # Keep backing length fixed so every alias keeps its range.
        memoryview(s.a)[s.o:s.o + s.l] = bytes(s.l)
        return
    for i in range(s.l):
        s.a[s.o + i] = zero()
