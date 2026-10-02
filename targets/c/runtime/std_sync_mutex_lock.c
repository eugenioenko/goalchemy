/* std.sync.mutex.lock: FIFO handoff to waiting tasks. */
#include "gx.h"

typedef struct cleanup_arg {
    gx_Mutex *object;
    gx_Task *task;
} cleanup_arg;
static void cleanup(void *arg) {
    cleanup_arg *a = arg;
    if (!a->task || !gx_owner_current(a->task->owner))
        return;
    gx_Mutex *o = a->object;
    size_t n = 0, old = o->n;
    for (size_t i = 0; i < old; i++)
        if (o->waiters[i] != a->task)
            o->waiters[n++] = o->waiters[i];
    if (n < old)
        memset(o->waiters + n, 0, (old - n) * sizeof(gx_Task *));
    o->n = n;
    a->object = NULL;
    a->task = NULL;
}

void gx_std_sync_mutex_lock(gx_Task *t, gx_V mv) {
    gx_owner_check(t->owner);
    gx_set_rv(t, 0, NULL);
    gx_Mutex *m = gx_nilchk(mv).u.p;
    if (!m->locked) {
        m->locked = true;
        return;
    }
    if (m->n == m->cap) {
        size_t c = m->cap ? m->cap * 2 : 4;
        gx_Task **w = GC_MALLOC(c * sizeof(gx_Task *));
        if (m->n)
            memcpy(w, m->waiters, m->n * sizeof(gx_Task *));
        m->waiters = w;
        m->cap = c;
    }
    m->waiters[m->n++] = t;
    cleanup_arg *a = GC_MALLOC(sizeof(*a));
    a->object = m;
    a->task = t;
    t->cleanup = cleanup;
    t->cleanup_arg = a;
    gx_block(t);
}
