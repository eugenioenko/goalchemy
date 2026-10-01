"""core.map.len: number of entries."""


def map_len(m):
    return 0 if m is None else len(m.index)
