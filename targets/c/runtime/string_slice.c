/* core.string.slice: s[lo:hi]; nil bounds use defaults. */
#include "gx.h"

gx_V gx_sslice(gx_V s, gx_V lo, gx_V hi, bool u) {
    int64_t l = lo.t == GX_NIL ? 0 : lo.u.i;
    int64_t h = hi.t == GX_NIL ? (int64_t)s.l : hi.u.i;
    gx_check2(l, h, s.l, "length", u);
    gx_V r = s;
    r.l = (uint32_t)(h - l);
    r.u.p = r.l ? (char *)s.u.p + l : NULL;
    return r;
}
