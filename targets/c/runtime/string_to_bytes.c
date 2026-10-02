/* core.string.to_bytes: independent native bytes, capacity equal to length. */
#include "gx.h"

gx_V gx_to_bytes(gx_V s) {
    uint8_t *a = gx_alloc_bytes(s.l);
    if (s.l) memcpy(a, s.u.p, s.l);
    return gx_byte_slice(a, s.l, s.l);
}
