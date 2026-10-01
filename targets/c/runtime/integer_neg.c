/* core.integer.neg: wrapping negation. */
#include "gx.h"

gx_V gx_neg_i8(gx_V a) { return gx_int(gx_w8(gx_wneg(a.u.i))); }
gx_V gx_neg_i16(gx_V a) { return gx_int(gx_w16(gx_wneg(a.u.i))); }
gx_V gx_neg_i32(gx_V a) { return gx_int(gx_w32(gx_wneg(a.u.i))); }
gx_V gx_neg_i64(gx_V a) { return gx_int(gx_w64(gx_wneg(a.u.i))); }
gx_V gx_neg_u8(gx_V a) { return gx_int(gx_wu8(gx_wneg(a.u.i))); }
gx_V gx_neg_u16(gx_V a) { return gx_int(gx_wu16(gx_wneg(a.u.i))); }
gx_V gx_neg_u32(gx_V a) { return gx_int(gx_wu32(gx_wneg(a.u.i))); }
gx_V gx_neg_u64(gx_V a) { return gx_int(gx_wu64(gx_wneg(a.u.i))); }
