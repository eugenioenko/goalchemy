/* core.slice.make: make([]T, len, cap) with zeroed capacity. */
#include "gx.h"

gx_V gx_make_slice(gx_V len, gx_V cap, gx_ZeroFn zero) {
    int64_t l = len.u.i, c = cap.u.i;
    if (l < 0 || l > (1LL << 53)) gx_runtime_panic("makeslice: len out of range");
    if (c < l || c > (1LL << 53)) gx_runtime_panic("makeslice: cap out of range");
    if (c > (int64_t)(UINT32_MAX / 2)) gx_fault("allocation exceeds host limits");
    gx_V *a = gx_alloc_vals((size_t)c);
    for (int64_t i = 0; i < c; i++) a[i] = zero();
    return gx_slice(a, (uint32_t)l, (uint32_t)c);
}
