/* core.integer.shl: n is a validated count (gx_count or gx_countu). */
#include "gx.h"

gx_V gx_shl_i8(gx_V a, unsigned n) {
    return gx_int(n >= 8 ? 0 : gx_w8((int64_t)((uint64_t)a.u.i << n)));
}

gx_V gx_shl_i16(gx_V a, unsigned n) {
    return gx_int(n >= 16 ? 0 : gx_w16((int64_t)((uint64_t)a.u.i << n)));
}

gx_V gx_shl_i32(gx_V a, unsigned n) {
    return gx_int(n >= 32 ? 0 : gx_w32((int64_t)((uint64_t)a.u.i << n)));
}

gx_V gx_shl_i64(gx_V a, unsigned n) {
    return gx_int(n >= 64 ? 0 : gx_w64((int64_t)((uint64_t)a.u.i << n)));
}

gx_V gx_shl_u8(gx_V a, unsigned n) {
    return gx_int(n >= 8 ? 0 : gx_wu8((int64_t)((uint64_t)a.u.i << n)));
}

gx_V gx_shl_u16(gx_V a, unsigned n) {
    return gx_int(n >= 16 ? 0 : gx_wu16((int64_t)((uint64_t)a.u.i << n)));
}

gx_V gx_shl_u32(gx_V a, unsigned n) {
    return gx_int(n >= 32 ? 0 : gx_wu32((int64_t)((uint64_t)a.u.i << n)));
}

gx_V gx_shl_u64(gx_V a, unsigned n) {
    return gx_int(n >= 64 ? 0 : gx_wu64((int64_t)((uint64_t)a.u.i << n)));
}

