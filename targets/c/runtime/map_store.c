/* core.map.store: assignment to a nil map panics. */
#include "gx.h"

void gx_map_set(gx_V m, gx_V k, gx_V v) {
    if (m.t == GX_NIL) gx_plain_panic("assignment to entry in nil map");
    gx_Map *x = m.u.p;
    gx_Buf key = {0};
    x->key_of(k, &key);
    gx_Entry *e = gx_map_find(x, key.b, key.n);
    if (e) {
        e->v = v;
        return;
    }
    gx_map_insert(x, k, v, key.b, key.n);
}
