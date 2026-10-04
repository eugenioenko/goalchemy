"""core.string.from_bytes: string(b), copying the bytes."""


def from_bytes(b):
    if b.a is None:
        return b""
    return bytes(memoryview(b.a)[b.o:b.o + b.l])
