/* core.string.decode_rune: the rune at a byte offset and its width. */
#include "gx.h"

gx_V gx_decode_rune(gx_V s, gx_V i) {
    size_t w;
    int64_t r = gx_utf8_decode(gx_sbytes(s), s.l, (size_t)i.u.i, &w);
    gx_V t[2] = {gx_int(r), gx_int((int64_t)w)};
    return gx_tuple(2, t);
}
