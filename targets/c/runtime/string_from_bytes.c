/* core.string.from_bytes: string(b) copies the bytes. */
#include "gx.h"

gx_V gx_from_bytes(gx_V b) {
    if (b.l == 0) return gx_str(NULL, 0);
    if (gx_byte_backing(b)) return gx_str((const char *)b.u.p, b.l);
    char *p = GC_MALLOC_ATOMIC(b.l);
    gx_V *e = gx_vals(b);
    for (uint32_t i = 0; i < b.l; i++) p[i] = (char)e[i].u.i;
    gx_V v = {0};
    v.t = GX_STR;
    v.l = b.l;
    v.u.p = p;
    return v;
}
