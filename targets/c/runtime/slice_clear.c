/* core.slice.clear: zeroes every element. */
#include "gx.h"

void gx_clear_slice(gx_V s, gx_ZeroFn zero) {
    if (gx_byte_backing(s)) { if (s.l) memset(s.u.p, 0, s.l); return; }
    for (uint32_t i = 0; i < s.l; i++) gx_vals(s)[i] = zero();
}
