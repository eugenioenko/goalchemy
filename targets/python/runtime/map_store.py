"""core.map.store: m[k] = v; storing into a nil map panics."""

from ..types.map import Entry
from ..types.panic import plain_panic


def map_set(m, k, v):
    if m is None:
        raise plain_panic("assignment to entry in nil map")
    key = m.key_of(k)
    e = m.index.get(key)
    if e is not None:
        e.v = v
        return
    e = Entry(k, v)
    m.index[key] = e
    m.entries.append(e)
