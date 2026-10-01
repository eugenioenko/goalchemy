/* core.map.delete: the key is encoded first, so unhashable keys panic even on nil maps. */
#include "gx.h"

void gx_map_delete(gx_V m, gx_V k, gx_KeyFn key_of) {
    gx_Buf key = {0};
    key_of(k, &key);
    if (m.t == GX_NIL) return;
    gx_map_remove(m.u.p, key.b, key.n);
}
