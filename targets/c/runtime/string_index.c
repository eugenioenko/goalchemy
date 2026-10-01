/* core.string.index: s[i] as a byte with bounds checking. */
#include "gx.h"

gx_V gx_sindex(gx_V s, gx_V i) { return gx_int(gx_sbytes(s)[gx_idx(i, s.l)]); }

gx_V gx_sindexu(gx_V s, gx_V i) { return gx_int(gx_sbytes(s)[gx_idxu(i, s.l)]); }
