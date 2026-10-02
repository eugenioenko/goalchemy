/* core.slice.store: s[i] = v with bounds checking. */
#include "gx.h"

void gx_sset(gx_V s, gx_V i, gx_V v) { size_t k = gx_idx(i, s.l); if (gx_byte_backing(s)) gx_bytes(s)[k] = (uint8_t)v.u.i; else gx_vals(s)[k] = v; }

void gx_ssetu(gx_V s, gx_V i, gx_V v) { size_t k = gx_idxu(i, s.l); if (gx_byte_backing(s)) gx_bytes(s)[k] = (uint8_t)v.u.i; else gx_vals(s)[k] = v; }
