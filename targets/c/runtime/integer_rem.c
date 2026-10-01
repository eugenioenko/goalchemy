/* core.integer.rem: truncated remainder; division by zero panics. */
#include "gx.h"

gx_V gx_rem_i8(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    if (y == 0) gx_div_zero();
    return gx_int(gx_w8(x % y));
}

gx_V gx_rem_i16(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    if (y == 0) gx_div_zero();
    return gx_int(gx_w16(x % y));
}

gx_V gx_rem_i32(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    if (y == 0) gx_div_zero();
    return gx_int(gx_w32(x % y));
}

gx_V gx_rem_i64(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    if (y == 0) gx_div_zero();
    if (y == -1) return gx_int(0);
    return gx_int(gx_w64(x % y));
}

gx_V gx_rem_u8(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    if (y == 0) gx_div_zero();
    return gx_int(gx_wu8(x % y));
}

gx_V gx_rem_u16(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    if (y == 0) gx_div_zero();
    return gx_int(gx_wu16(x % y));
}

gx_V gx_rem_u32(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    if (y == 0) gx_div_zero();
    return gx_int(gx_wu32(x % y));
}

gx_V gx_rem_u64(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    if (y == 0) gx_div_zero();
    return gx_int((int64_t)((uint64_t)x % (uint64_t)y));
}

