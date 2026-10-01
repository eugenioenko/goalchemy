/* core.integer.not: bitwise complement. */
#include "gx.h"

gx_V gx_not_i8(gx_V a) { return gx_int(gx_w8(~a.u.i)); }
gx_V gx_not_i16(gx_V a) { return gx_int(gx_w16(~a.u.i)); }
gx_V gx_not_i32(gx_V a) { return gx_int(gx_w32(~a.u.i)); }
gx_V gx_not_i64(gx_V a) { return gx_int(gx_w64(~a.u.i)); }
gx_V gx_not_u8(gx_V a) { return gx_int(gx_wu8(~a.u.i)); }
gx_V gx_not_u16(gx_V a) { return gx_int(gx_wu16(~a.u.i)); }
gx_V gx_not_u32(gx_V a) { return gx_int(gx_wu32(~a.u.i)); }
gx_V gx_not_u64(gx_V a) { return gx_int(gx_wu64(~a.u.i)); }
