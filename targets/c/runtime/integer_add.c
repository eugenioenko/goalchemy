/* core.integer.add: wrapping addition. */
#include "gx.h"

gx_V gx_add_i8(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_w8(gx_wadd(x, y)));
}

gx_V gx_add_i16(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_w16(gx_wadd(x, y)));
}

gx_V gx_add_i32(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_w32(gx_wadd(x, y)));
}

gx_V gx_add_i64(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_w64(gx_wadd(x, y)));
}

gx_V gx_add_u8(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_wu8(gx_wadd(x, y)));
}

gx_V gx_add_u16(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_wu16(gx_wadd(x, y)));
}

gx_V gx_add_u32(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_wu32(gx_wadd(x, y)));
}

gx_V gx_add_u64(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_wu64(gx_wadd(x, y)));
}

