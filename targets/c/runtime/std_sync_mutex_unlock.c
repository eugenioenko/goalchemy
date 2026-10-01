/* std.sync.mutex.unlock. */
#include "gx.h"

void gx_std_sync_mutex_unlock(gx_V mv) {
    gx_Mutex *m = gx_nilchk(mv).u.p;
    if (!m->locked) gx_fatal("sync: unlock of unlocked mutex");
    if (m->n == 0) {
        m->locked = false;
        return;
    }
    gx_Task *w = m->waiters[0];
    memmove(m->waiters, m->waiters + 1, (m->n - 1) * sizeof(gx_Task *));
    m->n--;
    gx_ready(w);
}
