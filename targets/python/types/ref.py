"""Pointers to non-aggregate storage. Pointers to structs and arrays are the
stable host objects themselves."""

import weakref


class Cell:
    """Heap storage for a boxed variable or new(T)."""

    __slots__ = ("v", "__weakref__")

    def __init__(self, v):
        self.v = v


class FieldRef:
    """A pointer to a struct field holding a non-aggregate value."""

    __slots__ = ("o", "k", "__weakref__")

    def __init__(self, o, k):
        self.o = o
        self.k = k

    @property
    def v(self):
        return getattr(self.o, self.k)

    @v.setter
    def v(self, x):
        setattr(self.o, self.k, x)


_refs = weakref.WeakKeyDictionary()


def field_ref(o, k):
    """Returns the canonical pointer to o's field k, so pointer equality holds."""
    m = _refs.get(o)
    if m is None:
        m = {}
        _refs[o] = m
    r = m.get(k)
    if r is None:
        r = FieldRef(o, k)
        m[k] = r
    return r
