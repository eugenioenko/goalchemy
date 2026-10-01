/* core.select: chooses uniformly among ready cases with the shared choice
 * source; rv is (index, value, ok) with index -1 for default. */
#include "gx.h"

typedef struct cleanup_arg {
    int n;
    gx_Case *cases;
    bool *st;
} cleanup_arg;

static void remove_sel(gx_WaitQ *q, bool *st) {
    gx_Waiter *prev = NULL;
    for (gx_Waiter *w = q->head; w;) {
        gx_Waiter *nx = w->next;
        if (w->sel == st) {
            if (prev)
                prev->next = nx;
            else
                q->head = nx;
            if (q->tail == w) q->tail = prev;
        } else {
            prev = w;
        }
        w = nx;
    }
}

static void cleanup(void *p) {
    cleanup_arg *a = p;
    for (int i = 0; i < a->n; i++) {
        if (a->cases[i].ch.t == GX_NIL) continue;
        gx_Chan *c = a->cases[i].ch.u.p;
        remove_sel(&c->sendq, a->st);
        remove_sel(&c->recvq, a->st);
    }
}

void gx_select(gx_Task *t, bool has_default, int n, const gx_Case *cases) {
    int *ready = GC_MALLOC_ATOMIC((size_t)(n ? n : 1) * sizeof(int));
    int nr = 0;
    for (int i = 0; i < n; i++) {
        if (cases[i].ch.t == GX_NIL) continue;
        gx_Chan *c = cases[i].ch.u.p;
        bool ok = cases[i].send ? (c->closed || gx_has_live(&c->recvq) || c->len < c->size)
                                : (c->len > 0 || gx_has_live(&c->sendq) || c->closed);
        if (ok) ready[nr++] = i;
    }
    if (nr > 0) {
        int i = ready[gx_choose((size_t)nr)];
        gx_Chan *c = cases[i].ch.u.p;
        if (cases[i].send) {
            if (c->closed) gx_plain_panic("send on closed channel");
            gx_Waiter *w = gx_dequeue(&c->recvq);
            if (w) {
                gx_recv_done(w, cases[i].v, true);
            } else {
                c->buf[(c->head + c->len) % c->size] = cases[i].v;
                c->len++;
            }
            gx_V r[3] = {gx_int(i), gx_nil(), gx_bool(false)};
            gx_set_rv(t, 3, r);
            return;
        }
        gx_V v;
        bool ok;
        gx_try_recv(c, &v, &ok);
        gx_V r[3] = {gx_int(i), v, gx_bool(ok)};
        gx_set_rv(t, 3, r);
        return;
    }
    if (has_default) {
        gx_V r[3] = {gx_int(-1), gx_nil(), gx_bool(false)};
        gx_set_rv(t, 3, r);
        return;
    }
    bool *st = GC_MALLOC_ATOMIC(sizeof(bool));
    *st = false;
    for (int i = 0; i < n; i++) {
        if (cases[i].ch.t == GX_NIL) continue;
        gx_Chan *c = cases[i].ch.u.p;
        gx_Waiter *w = GC_MALLOC(sizeof(gx_Waiter));
        w->task = t;
        w->val = cases[i].send ? cases[i].v : gx_nil();
        w->sel = st;
        w->idx = i;
        gx_enqueue(cases[i].send ? &c->sendq : &c->recvq, w);
    }
    gx_block(t);
    cleanup_arg *a = GC_MALLOC(sizeof(cleanup_arg));
    a->n = n;
    a->cases = GC_MALLOC((size_t)(n ? n : 1) * sizeof(gx_Case));
    if (n) memcpy(a->cases, cases, (size_t)n * sizeof(gx_Case));
    a->st = st;
    t->cleanup = cleanup;
    t->cleanup_arg = a;
}
