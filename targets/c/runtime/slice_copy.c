/* core.slice.copy: copy(dst, src) with overlap handled as Go does. */
#include "gx.h"

gx_V gx_copy(gx_V dst, gx_V src, gx_CloneFn clone) {
    uint32_t n = dst.l < src.l ? dst.l : src.l;
    if (n == 0) return gx_int(0);
    gx_V *tmp = gx_alloc_vals(n);
    memcpy(tmp, gx_vals(src), n * sizeof(gx_V));
    for (uint32_t i = 0; i < n; i++) gx_vals(dst)[i] = clone ? clone(tmp[i]) : tmp[i];
    return gx_int(n);
}

gx_V gx_copy_string(gx_V dst, gx_V src) {
    uint32_t n = dst.l < src.l ? dst.l : src.l;
    for (uint32_t i = 0; i < n; i++) gx_vals(dst)[i] = gx_int(gx_sbytes(src)[i]);
    return gx_int(n);
}
