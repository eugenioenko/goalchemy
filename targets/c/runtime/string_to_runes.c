/* core.string.to_runes: []rune(s) with capacity equal to length. */
#include "gx.h"

gx_V gx_to_runes(gx_V s) {
    size_t n = 0, w;
    for (size_t i = 0; i < s.l; i += w, n++) gx_utf8_decode(gx_sbytes(s), s.l, i, &w);
    gx_V *a = gx_alloc_vals(n);
    size_t k = 0;
    for (size_t i = 0; i < s.l; i += w) a[k++] = gx_int(gx_utf8_decode(gx_sbytes(s), s.l, i, &w));
    return gx_slice(a, (uint32_t)n, (uint32_t)n);
}
