/* core.chan.make: channels with a buffer and FIFO wait queues. */
#include "gx.h"

void gx_recv_done(gx_Waiter *w, gx_V v, bool ok) {
    if (w->sel) {
        *w->sel = true;
        gx_V r[3] = {gx_int(w->idx), v, gx_bool(ok)};
        gx_set_rv(w->task, 3, r);
    } else {
        gx_V r[2] = {v, gx_bool(ok)};
        gx_set_rv(w->task, 2, r);
    }
    gx_ready(w->task);
}

void gx_send_done(gx_Waiter *w, bool closed) {
    if (w->sel) {
        *w->sel = true;
        gx_V r[3] = {gx_int(w->idx), gx_nil(), gx_bool(false)};
        gx_set_rv(w->task, 3, r);
    } else {
        gx_set_rv(w->task, 0, NULL);
    }
    if (closed) w->task->resume_panic = gx_new_panic(gx_boxv(&GX_PLAIN_ERROR, gx_cstr("send on closed channel")));
    gx_ready(w->task);
}

static bool live(gx_Waiter *w) { return !w->sel || !*w->sel; }

gx_Waiter *gx_dequeue(gx_WaitQ *q) {
    while (q->head) {
        gx_Waiter *w = q->head;
        q->head = w->next;
        if (!q->head) q->tail = NULL;
        w->next = NULL;
        if (live(w)) return w;
    }
    return NULL;
}

bool gx_has_live(gx_WaitQ *q) {
    for (gx_Waiter *w = q->head; w; w = w->next)
        if (live(w)) return true;
    return false;
}

void gx_enqueue(gx_WaitQ *q, gx_Waiter *w) {
    w->next = NULL;
    if (q->tail)
        q->tail->next = w;
    else
        q->head = w;
    q->tail = w;
}

static void buf_push(gx_Chan *c, gx_V v) {
    c->buf[(c->head + c->len) % c->size] = v;
    c->len++;
}

/* Receives without blocking; returns whether the receive completed. */
bool gx_try_recv(gx_Chan *c, gx_V *v, bool *ok) {
    if (c->len > 0) {
        *v = c->buf[c->head];
        c->buf[c->head] = gx_nil();
        c->head = (c->head + 1) % c->size;
        c->len--;
        gx_Waiter *w = gx_dequeue(&c->sendq);
        if (w) {
            buf_push(c, w->val);
            gx_send_done(w, false);
        }
        *ok = true;
        return true;
    }
    gx_Waiter *w = gx_dequeue(&c->sendq);
    if (w) {
        *v = w->val;
        *ok = true;
        gx_send_done(w, false);
        return true;
    }
    if (c->closed) {
        *v = c->zero();
        *ok = false;
        return true;
    }
    return false;
}

gx_V gx_make_chan(gx_V size, gx_ZeroFn zero) {
    int64_t n = size.u.i;
    if (n < 0 || n > (1LL << 53)) gx_plain_panic("makechan: size out of range");
    if (n > (int64_t)(UINT32_MAX / 2)) gx_fault("channel buffer exceeds host limits");
    gx_Chan *c = GC_MALLOC(sizeof(gx_Chan));
    c->size = (size_t)n;
    c->buf = gx_alloc_vals(n ? (size_t)n : 1);
    c->zero = zero;
    return gx_obj(c);
}
