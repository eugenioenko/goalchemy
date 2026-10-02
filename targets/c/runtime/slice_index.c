/* core.slice.index: s[i] with bounds checking. */
#include "gx.h"

gx_V gx_sget(gx_V s, gx_V i) { size_t k = gx_idx(i, s.l); return gx_byte_backing(s) ? gx_int(gx_bytes(s)[k]) : gx_vals(s)[k]; }

gx_V gx_sgetu(gx_V s, gx_V i) { size_t k = gx_idxu(i, s.l); return gx_byte_backing(s) ? gx_int(gx_bytes(s)[k]) : gx_vals(s)[k]; }
