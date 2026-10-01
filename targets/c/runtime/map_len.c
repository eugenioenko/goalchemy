/* core.map.len. */
#include "gx.h"

gx_V gx_map_len(gx_V m) { return gx_int(m.t == GX_NIL ? 0 : (int64_t)((gx_Map *)m.u.p)->live); }
