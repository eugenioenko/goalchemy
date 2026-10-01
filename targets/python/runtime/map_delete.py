"""core.map.delete: delete(m, k); a nil map is a no-op after hashing."""


def map_delete(m, k, key_of):
    key = key_of(k)
    if m is None:
        return
    e = m.index.pop(key, None)
    if e is not None:
        e.live = False
        m.compact()
