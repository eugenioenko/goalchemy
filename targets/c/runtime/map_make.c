/* core.map.make. */
#include "gx.h"

gx_V gx_make_map(gx_KeyFn key_of) {
    gx_Map *m = GC_MALLOC(sizeof(gx_Map));
    m->key_of = key_of;
    return gx_obj(m);
}
