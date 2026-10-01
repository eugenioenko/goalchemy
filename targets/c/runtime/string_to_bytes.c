/* core.string.to_bytes: []byte(s) with capacity equal to length. */
#include "gx.h"

gx_V gx_to_bytes(gx_V s) {
    gx_V *a = gx_alloc_vals(s.l);
    for (uint32_t i = 0; i < s.l; i++) a[i] = gx_int(gx_sbytes(s)[i]);
    return gx_slice(a, s.l, s.l);
}
