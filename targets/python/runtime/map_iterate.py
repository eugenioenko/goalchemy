"""core.map.iterate: snapshot iteration in insertion order."""

from ..types.map import MapIter
from ..types.slice import Slice


def map_iter(m):
    return MapIter([] if m is None else list(m.entries))


def map_next(it):
    while it.i < len(it.entries):
        e = it.entries[it.i]
        it.i += 1
        if e.live:
            it.k = e.k
            it.v = e.v
            return True
    return False


def map_keys(m):
    it = map_iter(m)
    keys = []
    while map_next(it):
        keys.append(it.k)
    return Slice(keys, 0, len(keys), len(keys))
