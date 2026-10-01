/* core.map.clear. */
#include "gx.h"

void gx_map_clear(gx_V m) {
    if (m.t != GX_NIL) gx_map_reset(m.u.p);
}
