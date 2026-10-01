/* Tasks and resumable frames, and the scheduler state; scheduling
 * operations are in core.task.spawn. */
#include "gx.h"

#include <stdlib.h>

gx_Sched *gx_sched;

gx_Frame *gx_new_frame(int nl, gx_Step step, gx_Results results) {
    gx_Frame *f = GC_MALLOC(sizeof(gx_Frame));
    f->l = gx_alloc_vals((size_t)nl);
    f->nl = nl;
    f->step = step;
    f->results = results;
    return f;
}

gx_Task *gx_new_task(int id, gx_Frame *f) {
    gx_Task *t = GC_MALLOC(sizeof(gx_Task));
    t->id = id;
    t->frame = f;
    t->defer_target = -1;
    return t;
}

int64_t gx_seed(void) {
    const char *s = getenv("GOALCHEMY_SEED");
    if (!s) return 1;
    char *end;
    long long v = strtoll(s, &end, 10);
    if (*end || v <= 0 || v >= (1LL << 32)) return 1;
    return v;
}

gx_Sched *gx_new_sched(gx_Task *main, bool harness) {
    gx_Sched *s = GC_MALLOC(sizeof(gx_Sched));
    s->cur = main;
    s->next_id = 1;
    s->rng = gx_seed();
    s->harness = harness;
    return s;
}

gx_Task *gx_cur_task(void) {
    if (!gx_sched) gx_sched = gx_new_sched(gx_new_task(0, NULL), false);
    return gx_sched->cur;
}

void gx_set_rv(gx_Task *t, int n, const gx_V *vs) {
    t->rv = gx_alloc_vals((size_t)n);
    if (n) memcpy(t->rv, vs, (size_t)n * sizeof(gx_V));
    t->nrv = n;
}
