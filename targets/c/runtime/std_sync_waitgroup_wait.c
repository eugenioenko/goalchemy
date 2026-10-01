/* std.sync.waitgroup.wait: a pause primitive. */
#include "gx.h"

void gx_std_sync_waitgroup_wait(gx_Task *t, gx_V wv) {
    gx_set_rv(t, 0, NULL);
    gx_WaitGroup *wg = gx_nilchk(wv).u.p;
    if (wg->n == 0) return;
    if (wg->nw == wg->cap) {
        size_t c = wg->cap ? wg->cap * 2 : 4;
        gx_Task **w = GC_MALLOC(c * sizeof(gx_Task *));
        if (wg->nw) memcpy(w, wg->waiters, wg->nw * sizeof(gx_Task *));
        wg->waiters = w;
        wg->cap = c;
    }
    wg->waiters[wg->nw++] = t;
    gx_block(t);
}
