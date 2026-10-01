"""core.chan.len: buffered element count."""


def chan_len(ch):
    return 0 if ch is None else len(ch.buf)
