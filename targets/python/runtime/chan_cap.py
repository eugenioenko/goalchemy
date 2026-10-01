"""core.chan.cap: buffer capacity."""


def chan_cap(ch):
    return 0 if ch is None else ch.size
