"""Interface values, type descriptors, and dynamic dispatch."""

_next_type_id = [1]


class TypeDesc:
    __slots__ = ("id", "name", "kind", "eq", "key", "methods", "basic", "comparable")

    def __init__(self, name, kind, eq, key, methods, basic=None, comparable=True):
        self.id = _next_type_id[0]
        _next_type_id[0] += 1
        self.name = name
        self.kind = kind
        self.eq = eq
        self.key = key
        self.methods = methods
        self.basic = basic
        self.comparable = comparable


def type_desc(**kw):
    return TypeDesc(**kw)


class Box:
    """A non-nil interface value: dynamic type and value."""

    __slots__ = ("t", "v")

    def __init__(self, t, v):
        self.t = t
        self.v = v


def box(t, v):
    return Box(t, v)


def iface_eq(a, b):
    if a is None or b is None:
        return a is b
    if a.t is not b.t:
        return False
    return a.t.eq(a.v, b.v)


def iface_key(a):
    if a is None:
        return ("nil",)
    return (a.t.id, a.t.key(a.v))


def implements_all(t, ids):
    for i in ids:
        if i not in t.methods:
            return i
    return None
