"""core.integer.compare: three-way comparison."""


def compare_int(a, b):
    return -1 if a < b else 1 if a > b else 0
