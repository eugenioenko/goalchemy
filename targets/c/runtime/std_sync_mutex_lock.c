/* std.sync.mutex.lock: FIFO handoff to waiting tasks. */
#include "gx.h"

void gx_std_sync_mutex_lock(gx_Task *t, gx_V mv) {
    gx_set_rv(t, 0, NULL);
    gx_Mutex *m = gx_nilchk(mv).u.p;
    if (!m->locked) {
        m->locked = true;
        return;
    }
    if (m->n == m->cap) {
        size_t c = m->cap ? m->cap * 2 : 4;
        gx_Task **w = GC_MALLOC(c * sizeof(gx_Task *));
        if (m->n) memcpy(w, m->waiters, m->n * sizeof(gx_Task *));
        m->waiters = w;
        m->cap = c;
    }
    m->waiters[m->n++] = t;
    gx_block(t);
}
