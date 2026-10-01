/* core.string.concat. */
#include "gx.h"

gx_V gx_concat(gx_V a, gx_V b) {
    if (b.l == 0) return a;
    if (a.l == 0) return b;
    size_t n = (size_t)a.l + b.l;
    char *p = GC_MALLOC_ATOMIC(n);
    memcpy(p, a.u.p, a.l);
    memcpy(p + a.l, b.u.p, b.l);
    gx_V v = {0};
    v.t = GX_STR;
    v.l = (uint32_t)n;
    v.u.p = p;
    return v;
}
