/* core.map.iterate: insertion-ordered snapshot iteration that skips deleted entries. */
#include "gx.h"

gx_V gx_map_iter(gx_V m) {
    gx_MapIter *it = GC_MALLOC(sizeof(gx_MapIter));
    if (m.t != GX_NIL) {
        gx_Map *x = m.u.p;
        it->n = x->n;
        it->entries = GC_MALLOC((x->n ? x->n : 1) * sizeof(gx_Entry *));
        if (x->n) memcpy(it->entries, x->entries, x->n * sizeof(gx_Entry *));
    }
    return gx_obj(it);
}

gx_V gx_map_next(gx_V v) {
    gx_MapIter *it = v.u.p;
    while (it->i < it->n) {
        gx_Entry *e = it->entries[it->i++];
        if (e->live) {
            it->k = e->k;
            it->v = e->v;
            return gx_bool(true);
        }
    }
    return gx_bool(false);
}

gx_V gx_iter_key(gx_V v) { return ((gx_MapIter *)v.u.p)->k; }

gx_V gx_iter_val(gx_V v) { return ((gx_MapIter *)v.u.p)->v; }

gx_V gx_map_keys(gx_V m) {
    gx_V it = gx_map_iter(m);
    gx_MapIter *x = it.u.p;
    gx_V *a = gx_alloc_vals(x->n);
    uint32_t n = 0;
    while (gx_b(gx_map_next(it))) a[n++] = x->k;
    return gx_slice(a, n, n);
}
