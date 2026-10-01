"""core.slice.make: make([]T, len, cap) with zeroed capacity."""

from ..types.slice import Slice
from ..types.panic import runtime_panic, fault

HOST_LIMIT = 2**32 - 1


def make_slice(n, c, zero):
    if n < 0 or n > 2**53:
        raise runtime_panic("makeslice: len out of range")
    if c < n or c > 2**53:
        raise runtime_panic("makeslice: cap out of range")
    if c > HOST_LIMIT:
        raise fault("allocation of %d elements exceeds host limits" % c)
    return Slice([zero() for _ in range(c)], 0, n, c)
