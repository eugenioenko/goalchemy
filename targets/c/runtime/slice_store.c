/* core.slice.store: s[i] = v with bounds checking. */
#include "gx.h"

void gx_sset(gx_V s, gx_V i, gx_V v) { gx_vals(s)[gx_idx(i, s.l)] = v; }

void gx_ssetu(gx_V s, gx_V i, gx_V v) { gx_vals(s)[gx_idxu(i, s.l)] = v; }
