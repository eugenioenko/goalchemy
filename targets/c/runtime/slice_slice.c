/* core.slice.slice: s[lo:hi:max] sharing backing storage; nil bounds use defaults. */
#include "gx.h"

static void bounds(gx_V lo, gx_V hi, gx_V max, int64_t len, int64_t cap, const char *word, bool u, int64_t *l, int64_t *h, int64_t *m) {
    *l = lo.t == GX_NIL ? 0 : lo.u.i;
    *h = hi.t == GX_NIL ? len : hi.u.i;
    if (max.t == GX_NIL) {
        gx_check2(*l, *h, cap, word, u);
        *m = cap;
    } else {
        *m = max.u.i;
        gx_check3(*l, *h, *m, cap, word, u);
    }
}

gx_V gx_reslice(gx_V s, gx_V lo, gx_V hi, gx_V max, bool u) {
    int64_t l, h, m;
    bounds(lo, hi, max, s.l, s.c, "capacity", u, &l, &h, &m);
    if (!s.u.p) return s;
    if (gx_byte_backing(s)) return gx_byte_slice(gx_bytes(s) + l, (uint32_t)(h - l), (uint32_t)(m - l));
    return gx_slice(gx_vals(s) + l, (uint32_t)(h - l), (uint32_t)(m - l));
}

/* Slices an array of length n through a pointer: (&a)[lo:hi:max]. */
gx_V gx_slice_array(gx_V a, size_t n, gx_V lo, gx_V hi, gx_V max, bool u) {
    gx_nilchk(a);
    int64_t l, h, m;
    bounds(lo, hi, max, (int64_t)n, (int64_t)n, "length", u, &l, &h, &m);
    if (gx_byte_backing(a)) return gx_byte_slice(gx_bytes(a) + l, (uint32_t)(h - l), (uint32_t)(m - l));
    return gx_slice(gx_vals(a) + l, (uint32_t)(h - l), (uint32_t)(m - l));
}
