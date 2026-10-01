"""core.string.compare: bytewise comparison."""


def scompare(a, b):
    return -1 if a < b else 1 if a > b else 0
