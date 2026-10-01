/* core.chan.close: wakes every receiver and panics blocked senders. */
#include "gx.h"

void gx_chan_close(gx_V ch) {
    if (ch.t == GX_NIL) gx_plain_panic("close of nil channel");
    gx_Chan *c = ch.u.p;
    if (c->closed) gx_plain_panic("close of closed channel");
    c->closed = true;
    for (gx_Waiter *w = gx_dequeue(&c->recvq); w; w = gx_dequeue(&c->recvq)) gx_recv_done(w, c->zero(), false);
    for (gx_Waiter *w = gx_dequeue(&c->sendq); w; w = gx_dequeue(&c->sendq)) gx_send_done(w, true);
}
