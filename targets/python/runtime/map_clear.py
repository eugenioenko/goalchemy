"""core.map.clear: remove every entry."""


def map_clear(m):
    if m is None:
        return
    for e in m.entries:
        e.live = False
    m.index.clear()
    m.entries = []
