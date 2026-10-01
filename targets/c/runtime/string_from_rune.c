/* core.string.from_rune: string(r) encodes one code point. */
#include "gx.h"

gx_V gx_from_rune(gx_V r) {
    gx_Buf b = {0};
    gx_utf8_encode(r.u.i, &b);
    return gx_str((const char *)b.b, b.n);
}

/* For unsigned 64-bit operands: negative values are huge. */
gx_V gx_from_rune_u(gx_V r) { return gx_from_rune(gx_int(r.u.i < 0 ? -1 : r.u.i)); }
