/* std.sync.waitgroup.wait: a pause primitive. */
#include "gx.h"

typedef struct cleanup_arg {
    gx_WaitGroup *object;
    gx_Task *task;
} cleanup_arg;
static void cleanup(void *arg) {
    cleanup_arg *a = arg;
    if (!a->task || !gx_owner_current(a->task->owner))
        return;
    gx_WaitGroup *o = a->object;
    size_t n = 0, old = o->nw;
    for (size_t i = 0; i < old; i++)
        if (o->waiters[i] != a->task)
            o->waiters[n++] = o->waiters[i];
    if (n < old)
        memset(o->waiters + n, 0, (old - n) * sizeof(gx_Task *));
    o->nw = n;
    a->object = NULL;
    a->task = NULL;
}

void gx_std_sync_waitgroup_wait(gx_Task *t, gx_V wv) {
    gx_owner_check(t->owner);
    gx_set_rv(t, 0, NULL);
    gx_WaitGroup *wg = gx_nilchk(wv).u.p;
    if (wg->n == 0)
        return;
    if (wg->nw == wg->cap) {
        size_t c = wg->cap ? wg->cap * 2 : 4;
        gx_Task **w = GC_MALLOC(c * sizeof(gx_Task *));
        if (wg->nw)
            memcpy(w, wg->waiters, wg->nw * sizeof(gx_Task *));
        wg->waiters = w;
        wg->cap = c;
    }
    wg->waiters[wg->nw++] = t;
    cleanup_arg *a = GC_MALLOC(sizeof(*a));
    a->object = wg;
    a->task = t;
    t->cleanup = cleanup;
    t->cleanup_arg = a;
    gx_block(t);
}
