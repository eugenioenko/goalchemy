"""core.slice.clear: zero the elements below the length."""


def clear_slice(s, zero):
    for i in range(s.l):
        s.a[s.o + i] = zero()
