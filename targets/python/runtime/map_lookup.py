"""core.map.lookup: v, ok := m[k]. Nil maps still hash the key."""


def map_get(m, k, key_of, zero):
    key = key_of(k)
    if m is None:
        return zero(), False
    e = m.index.get(key)
    if e is None:
        return zero(), False
    return e.v, True
