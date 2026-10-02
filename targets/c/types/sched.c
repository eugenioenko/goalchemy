/* Tasks and resumable frames, and the scheduler state; scheduling
 * operations are in core.task.spawn. */
#include "gx.h"

#include <stdlib.h>
#include <stdatomic.h>
#include <stdio.h>

gx_Sched *gx_sched;
size_t gx_source_depth;
static atomic_bool gx_entry_active;

bool gx_entry_reserve(void) {
    bool expected = false;
    return atomic_compare_exchange_strong(&gx_entry_active, &expected, true);
}
void gx_entry_release(void) { atomic_store(&gx_entry_active, false); }
bool gx_owner_current(gx_Sched *s) {
    return s && s == gx_sched && !s->retired && pthread_equal(s->thread, pthread_self());
}
void gx_owner_check(gx_Sched *s) {
    if (!gx_owner_current(s) || s->retiring)
        gx_host_fault("source operation on a foreign or retired owner");
}
_Noreturn void gx_host_fault(const char *message) {
    gx_Sched *s = gx_sched;
    if (s && s->host && s->escape && pthread_equal(s->thread, pthread_self())) {
        if (!s->fault)
            s->fault = GC_strdup(message);
        gx_handler = NULL;
        gx_source_depth = 0;
        longjmp(*s->escape, 1);
    }
    /* A foreign native caller receives no permission to unwind source stacks. */
    fprintf(stderr, "goalchemy host fault: %s\n", message);
    abort();
}
void gx_source_enter(void) {
    if (!gx_sched || !gx_sched->host)
        return;
    gx_owner_check(gx_sched);
    if (gx_source_depth >= 256) {
        gx_sched->fatal = "stack overflow";
        gx_handler = NULL;
        gx_source_depth = 0;
        longjmp(*gx_sched->escape, 1);
    }
    gx_source_depth++;
}
void gx_source_leave(void) {
    if (gx_sched && gx_sched->host && gx_source_depth)
        gx_source_depth--;
}
void gx_own_task(gx_Sched *s, gx_Task *t) {
    t->owner = s;
    t->next_all = s->all;
    s->all = t;
}
void gx_retire(gx_Sched *s) {
    if (!s || s->retired)
        return;
    if (s->retire) {
        s->retire(s);
        return;
    }
    /* A lazy sequential owner has no native operations, but can own contexts. */
    s->retiring = true;
    for (gx_Task *t = s->all; t; t = t->next_all) {
        if (t->cleanup) {
            void (*cleanup)(void *) = t->cleanup;
            void *arg = t->cleanup_arg;
            t->cleanup = NULL;
            t->cleanup_arg = NULL;
            gx_Handler h;
            if (!GX_TRY(h)) {
                cleanup(arg);
                GX_END(h);
            }
        }
        t->frame = NULL;
        t->rv = NULL;
        t->nrv = 0;
        t->resume_panic = t->cur_panic = NULL;
        t->done = true;
    }
    gx_Task *task = s->all;
    s->all = NULL;
    while (task) {
        gx_Task *next = task->next_all;
        task->next_all = NULL;
        task = next;
    }
    for (gx_Context *c = s->contexts; c; c = c->next_owner) {
        c->parent = NULL;
        c->children = NULL;
        c->n = c->cap = 0;
        c->done = c->err = gx_nil();
    }
    gx_Context *context = s->contexts;
    s->contexts = NULL;
    while (context) {
        gx_Context *next = context->next_owner;
        context->next_owner = NULL;
        context = next;
    }
    s->cur = NULL;
    s->runq = NULL;
    s->qhead = s->qlen = s->qcap = 0;
    s->timers = NULL;
    s->ntimers = s->tcap = 0;
    s->retiring = false;
    s->retired = true;
}

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
    if (!s)
        return 1;
    char *end;
    long long v = strtoll(s, &end, 10);
    if (*end || v <= 0 || v >= (1LL << 32))
        return 1;
    return v;
}

gx_Sched *gx_new_sched(gx_Task *main, bool harness) {
    gx_Sched *s = GC_MALLOC(sizeof(gx_Sched));
    s->thread = pthread_self();
    s->cur = main;
    gx_own_task(s, main);
    s->next_id = 1;
    s->rng = gx_seed();
    s->harness = harness;
    return s;
}

gx_Task *gx_cur_task(void) {
    if (!gx_sched)
        gx_sched = gx_new_sched(gx_new_task(0, NULL), false);
    return gx_sched->cur;
}

void gx_set_rv(gx_Task *t, int n, const gx_V *vs) {
    if (t->owner) gx_owner_check(t->owner);
    t->rv = gx_alloc_vals((size_t)n);
    if (n)
        memcpy(t->rv, vs, (size_t)n * sizeof(gx_V));
    t->nrv = n;
}
