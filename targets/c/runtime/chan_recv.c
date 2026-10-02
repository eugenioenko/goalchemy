/* core.chan.recv: a pause primitive leaving (value, ok). */
#include "gx.h"

void gx_chan_recv(gx_Task *t, gx_V ch) {
    gx_owner_check(t->owner);
    if (ch.t == GX_NIL) {
        gx_block(t);
        return;
    }
    gx_Chan *c = ch.u.p;
    gx_V v;
    bool ok;
    if (gx_try_recv(c, &v, &ok)) {
        gx_V r[2] = {v, gx_bool(ok)};
        gx_set_rv(t, 2, r);
        return;
    }
    gx_Waiter *w = GC_MALLOC(sizeof(gx_Waiter));
    w->task = t;
    gx_enqueue(&c->recvq, w);
    gx_block(t);
}
