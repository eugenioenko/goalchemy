/* core.integer.andnot: bit clear. */
#include "gx.h"

gx_V gx_andnot_i8(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_w8(x & ~y));
}

gx_V gx_andnot_i16(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_w16(x & ~y));
}

gx_V gx_andnot_i32(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_w32(x & ~y));
}

gx_V gx_andnot_i64(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_w64(x & ~y));
}

gx_V gx_andnot_u8(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_wu8(x & ~y));
}

gx_V gx_andnot_u16(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_wu16(x & ~y));
}

gx_V gx_andnot_u32(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_wu32(x & ~y));
}

gx_V gx_andnot_u64(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_wu64(x & ~y));
}

