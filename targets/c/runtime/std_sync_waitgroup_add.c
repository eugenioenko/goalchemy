/* std.sync.waitgroup.add. */
#include "gx.h"

void gx_std_sync_waitgroup_add(gx_V wv, gx_V d) {
    gx_cur_task(); gx_owner_check(gx_sched);
    gx_WaitGroup *wg = gx_nilchk(wv).u.p;
    wg->n += d.u.i;
    if (wg->n < 0) gx_throw(gx_boxv(&GX_STRING_TYPE, gx_cstr("sync: negative WaitGroup counter")));
    if (wg->n == 0) {
        size_t n = wg->nw;
        gx_Task **ws = wg->waiters;
        wg->waiters = NULL;
        wg->nw = wg->cap = 0;
        for (size_t i = 0; i < n; i++) gx_ready(ws[i]);
    }
}
