/* core.slice.index: s[i] with bounds checking. */
#include "gx.h"

gx_V gx_sget(gx_V s, gx_V i) { return gx_vals(s)[gx_idx(i, s.l)]; }

gx_V gx_sgetu(gx_V s, gx_V i) { return gx_vals(s)[gx_idxu(i, s.l)]; }
