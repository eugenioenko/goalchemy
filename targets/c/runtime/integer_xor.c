/* core.integer.xor: bitwise xor. */
#include "gx.h"

gx_V gx_xor_i8(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_w8(x ^ y));
}

gx_V gx_xor_i16(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_w16(x ^ y));
}

gx_V gx_xor_i32(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_w32(x ^ y));
}

gx_V gx_xor_i64(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_w64(x ^ y));
}

gx_V gx_xor_u8(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_wu8(x ^ y));
}

gx_V gx_xor_u16(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_wu16(x ^ y));
}

gx_V gx_xor_u32(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_wu32(x ^ y));
}

gx_V gx_xor_u64(gx_V a, gx_V b) {
    int64_t x = a.u.i, y = b.u.i;
    return gx_int(gx_wu64(x ^ y));
}

