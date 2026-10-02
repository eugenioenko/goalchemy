/* core.chan.send: a pause primitive. */
#include "gx.h"

void gx_chan_send(gx_Task *t, gx_V ch, gx_V v) {
    gx_owner_check(t->owner);
    gx_set_rv(t, 0, NULL);
    if (ch.t == GX_NIL) {
        gx_block(t);
        return;
    }
    gx_Chan *c = ch.u.p;
    if (c->closed) gx_plain_panic("send on closed channel");
    gx_Waiter *r = gx_dequeue(&c->recvq);
    if (r) {
        gx_recv_done(r, v, true);
        return;
    }
    if (c->len < c->size) {
        c->buf[(c->head + c->len) % c->size] = v;
        c->len++;
        return;
    }
    gx_Waiter *w = GC_MALLOC(sizeof(gx_Waiter));
    w->task = t;
    w->val = v;
    gx_enqueue(&c->sendq, w);
    gx_block(t);
}
