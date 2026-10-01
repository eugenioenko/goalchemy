/* core.slice.to_array: [N]T(s) copies the first N elements. */
#include "gx.h"

#include <stdio.h>

gx_V gx_slice_to_array(gx_V s, size_t n, gx_CloneFn clone) {
    if (s.l < n) {
        char buf[160];
        snprintf(buf, sizeof buf, "cannot convert slice with length %u to array or pointer to array with length %zu", s.l, n);
        gx_runtime_panic(buf);
    }
    gx_V *a = gx_alloc_vals(n);
    for (size_t i = 0; i < n; i++) a[i] = clone ? clone(gx_vals(s)[i]) : gx_vals(s)[i];
    return gx_obj(a);
}
