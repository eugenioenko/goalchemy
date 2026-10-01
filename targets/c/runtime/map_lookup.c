/* core.map.lookup: returns (value, ok). */
#include "gx.h"

gx_V gx_map_get(gx_V m, gx_V k, gx_KeyFn key_of, gx_ZeroFn zero) {
    gx_Buf key = {0};
    key_of(k, &key);
    gx_Entry *e = m.t == GX_NIL ? NULL : gx_map_find(m.u.p, key.b, key.n);
    gx_V r[2] = {e ? e->v : zero(), gx_bool(e != NULL)};
    return gx_tuple(2, r);
}
