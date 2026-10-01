"""Slice headers over shared backing lists. Headers are immutable values."""


class Slice:
    __slots__ = ("a", "o", "l", "c")

    def __init__(self, a, o, l, c):
        self.a = a
        self.o = o
        self.l = l
        self.c = c


NIL = Slice(None, 0, 0, 0)


def from_list(a):
    return Slice(a, 0, len(a), len(a))
