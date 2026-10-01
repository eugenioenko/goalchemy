/* core.string.from_runes: string(runes) encodes each code point. */
#include "gx.h"

gx_V gx_from_runes(gx_V r) {
    gx_Buf b = {0};
    for (uint32_t i = 0; i < r.l; i++) gx_utf8_encode(gx_vals(r)[i].u.i, &b);
    return gx_str((const char *)b.b, b.n);
}
