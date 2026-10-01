/* core.integer.shr: arithmetic for signed kinds, logical for unsigned. */
#include "gx.h"

gx_V gx_shr_i8(gx_V a, unsigned n) {
    int64_t x = a.u.i;
    return gx_int(n >= 8 ? (x < 0 ? -1 : 0) : x >> n);
}

gx_V gx_shr_i16(gx_V a, unsigned n) {
    int64_t x = a.u.i;
    return gx_int(n >= 16 ? (x < 0 ? -1 : 0) : x >> n);
}

gx_V gx_shr_i32(gx_V a, unsigned n) {
    int64_t x = a.u.i;
    return gx_int(n >= 32 ? (x < 0 ? -1 : 0) : x >> n);
}

gx_V gx_shr_i64(gx_V a, unsigned n) {
    int64_t x = a.u.i;
    return gx_int(n >= 64 ? (x < 0 ? -1 : 0) : x >> n);
}

gx_V gx_shr_u8(gx_V a, unsigned n) {
    return gx_int(n >= 8 ? 0 : (int64_t)((uint64_t)a.u.i >> n));
}

gx_V gx_shr_u16(gx_V a, unsigned n) {
    return gx_int(n >= 16 ? 0 : (int64_t)((uint64_t)a.u.i >> n));
}

gx_V gx_shr_u32(gx_V a, unsigned n) {
    return gx_int(n >= 32 ? 0 : (int64_t)((uint64_t)a.u.i >> n));
}

gx_V gx_shr_u64(gx_V a, unsigned n) {
    return gx_int(n >= 64 ? 0 : (int64_t)((uint64_t)a.u.i >> n));
}

