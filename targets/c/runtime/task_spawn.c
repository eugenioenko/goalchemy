/* core.task.spawn and the cooperative scheduler. A task is a stack of
 * frames driven by a trampoline; exactly one task runs at a time and
 * runnable tasks are dispatched in FIFO order. Pause primitives either
 * complete immediately, leaving their results in the task's rv, or block
 * the task until another task or a timer readies it. */
#include "gx.h"

#include <stdio.h>
#include <stdlib.h>

/* xorshift32 choice source, identical on every target. */
size_t gx_choose(size_t n) {
    int64_t x = gx_sched->rng;
    x ^= (x << 13) & 0xFFFFFFFFLL;
    x ^= (int64_t)((uint64_t)x >> 17);
    x ^= (x << 5) & 0xFFFFFFFFLL;
    gx_sched->rng = x;
    return (size_t)(x % (int64_t)n);
}

void gx_ready(gx_Task *t) {
    gx_Sched *s = gx_sched;
    if (s->qlen == s->qcap) {
        size_t c = s->qcap ? s->qcap * 2 : 16;
        gx_Task **q = GC_MALLOC(c * sizeof(gx_Task *));
        for (size_t i = 0; i < s->qlen; i++) q[i] = s->runq[(s->qhead + i) % s->qcap];
        s->runq = q;
        s->qcap = c;
        s->qhead = 0;
    }
    s->runq[(s->qhead + s->qlen) % s->qcap] = t;
    s->qlen++;
}

void gx_block(gx_Task *t) { t->blocked = true; }

void gx_add_timer(int64_t d, gx_Task *task, void (*f)(gx_V), gx_V arg) {
    gx_Sched *s = gx_sched;
    if (s->ntimers == s->tcap) {
        size_t c = s->tcap ? s->tcap * 2 : 8;
        gx_Timer *ts = GC_MALLOC(c * sizeof(gx_Timer));
        if (s->ntimers) memcpy(ts, s->timers, s->ntimers * sizeof(gx_Timer));
        s->timers = ts;
        s->tcap = c;
    }
    s->seq++;
    gx_Timer t = {s->clock + d, s->seq, task, f, arg};
    s->timers[s->ntimers++] = t;
}

_Noreturn void gx_fatal(const char *msg) {
    char buf[256];
    int n = snprintf(buf, sizeof buf, "fatal error: %s\n", msg);
    gx_stderr(buf, (size_t)n);
    exit(2);
}

static void fire_timers(void) {
    gx_Sched *s = gx_sched;
    int64_t at = INT64_MAX;
    for (size_t i = 0; i < s->ntimers; i++)
        if (s->timers[i].at < at) at = s->timers[i].at;
    s->clock = at;
    size_t nd = 0, nk = 0;
    gx_Timer *due = GC_MALLOC(s->ntimers * sizeof(gx_Timer));
    for (size_t i = 0; i < s->ntimers; i++) {
        if (s->timers[i].at == at)
            due[nd++] = s->timers[i];
        else
            s->timers[nk++] = s->timers[i];
    }
    s->ntimers = nk;
    for (size_t i = 1; i < nd; i++)
        for (size_t j = i; j > 0 && due[j - 1].seq > due[j].seq; j--) {
            gx_Timer t = due[j];
            due[j] = due[j - 1];
            due[j - 1] = t;
        }
    for (size_t i = 0; i < nd; i++) {
        if (due[i].f) due[i].f(due[i].arg);
        if (due[i].task) gx_ready(due[i].task);
    }
}

static gx_Task *next(void) {
    gx_Sched *s = gx_sched;
    while (s->qlen == 0) {
        if (s->ntimers == 0) {
            if (s->harness) {
                gx_blocked_signal = 1;
                gx_raise(gx_new_panic(gx_nil()));
            }
            gx_fatal("all goroutines are asleep - deadlock!");
        }
        fire_timers();
    }
    gx_Task *t = s->runq[s->qhead];
    s->qhead = (s->qhead + 1) % s->qcap;
    s->qlen--;
    return t;
}

static void exit_frame(gx_Task *t, gx_Frame *f, gx_Panic *p);
static void defer_runner_step(gx_Task *t, gx_Frame *r);

static void run(gx_Task *t) {
    gx_sched->cur = t;
    t->blocked = false;
    if (t->cleanup) {
        void (*c)(void *) = t->cleanup;
        t->cleanup = NULL;
        c(t->cleanup_arg);
    }
    while (!t->blocked && t->frame) {
        gx_Panic *rp = t->resume_panic;
        if (rp) {
            t->resume_panic = NULL;
            exit_frame(t, t->frame, rp);
            continue;
        }
        gx_Frame *volatile f = t->frame;
        gx_Handler h;
        if (GX_TRY(h)) {
            exit_frame(t, f, gx_thrown);
            continue;
        }
        f->step(t, f);
        GX_END(h);
    }
}

static void finish(gx_Task *t, gx_Frame *f);

static void exit_frame(gx_Task *t, gx_Frame *f, gx_Panic *p) {
    if (p) {
        gx_chain_panic(p, f->panicking);
        f->panicking = p;
    }
    t->frame = f;
    if (f->defers) {
        gx_Frame *r = gx_new_frame(2, defer_runner_step, NULL);
        r->a = f;
        r->parent = f;
        t->frame = r;
        return;
    }
    finish(t, f);
}

static void after(gx_Frame *tf, gx_Panic *p) {
    if (p) {
        gx_chain_panic(p, tf->panicking);
        tf->panicking = p;
        return;
    }
    if (tf->panicking && tf->panicking->recovered) tf->panicking = NULL;
}

static void restore(gx_Task *t, gx_Frame *r) {
    t->cur_panic = r->l[0].u.p;
    t->defer_target = r->l[1].u.i;
}

static void child_done(gx_Task *t, gx_Frame *r, gx_Panic *p) {
    restore(t, r);
    r->b = NULL;
    t->frame = r;
    after(r->a, p);
}

static void finish(gx_Task *t, gx_Frame *f) {
    gx_Panic *p = f->panicking;
    gx_Frame *parent = f->parent;
    t->frame = parent;
    if (!parent) {
        t->done = true;
        if (p) {
            if (gx_sched->harness) gx_raise(p);
            gx_report_panic(p);
        }
        return;
    }
    if (parent->step == defer_runner_step && parent->b == f) {
        child_done(t, parent, p);
        return;
    }
    if (p) {
        exit_frame(t, parent, p);
        return;
    }
    gx_V out[16];
    int n = f->results ? f->results(f, out) : 0;
    gx_set_rv(t, n, out);
}

static void defer_runner_step(gx_Task *t, gx_Frame *r) {
    gx_Frame *tf = r->a;
    while (tf->defers) {
        gx_Deferred *d = tf->defers;
        tf->defers = d->next;
        r->l[0] = gx_obj(t->cur_panic);
        r->l[1] = gx_int(t->defer_target);
        t->cur_panic = tf->panicking;
        t->defer_target = d->fid;
        gx_Handler h;
        if (GX_TRY(h)) {
            restore(t, r);
            after(tf, gx_thrown);
            continue;
        }
        gx_V c = gx_callv(d->f, d->n, d->args);
        GX_END(h);
        if (d->start) {
            gx_Frame *cf = gx_frameof(c);
            r->b = cf;
            cf->parent = r;
            t->frame = cf;
            return;
        }
        restore(t, r);
        after(tf, NULL);
    }
    t->frame = tf;
    finish(t, tf);
}

/* Pushes a callee frame: a pause point. */
void gx_call(gx_Task *t, gx_V child) {
    gx_Frame *c = gx_frameof(child);
    c->parent = t->frame;
    t->frame = c;
}

void gx_ret(gx_Task *t, gx_Frame *f) { exit_frame(t, f, NULL); }

static void sync_step(gx_Task *t, gx_Frame *f) {
    gx_V args = f->l[1];
    f->l[3] = gx_callv(f->l[0], (int)args.l, gx_vals(args));
    gx_ret(t, f);
}

static int sync_results(gx_Frame *f, gx_V *out) {
    int n = (int)f->l[2].u.i;
    if (n == 1) out[0] = f->l[3];
    for (int i = 0; n > 1 && i < n; i++) out[i] = gx_at(f->l[3], i);
    return n;
}

/* Runs an ordinary call of f with nres results as a frame. */
gx_V gx_sync_frame(gx_V f, int n, const gx_V *args, int nres) {
    gx_Frame *fr = gx_new_frame(4, sync_step, sync_results);
    fr->l[0] = f;
    fr->l[1] = gx_tuple(n, args);
    fr->l[2] = gx_int(nres);
    return gx_vframe(fr);
}

static gx_V adapt_code(gx_V *env, gx_V *args, int n) { return gx_sync_frame(env[0], n, args, (int)env[1].u.i); }

/* Adapts an ordinary function value with nres results to the resumable form. */
gx_V gx_adapt(gx_V f, int nres) {
    if (f.t == GX_NIL) return f;
    gx_V env[2] = {f, gx_int(nres)};
    return gx_func(gx_fid_of(f), adapt_code, 2, env);
}

gx_V gx_adapt_slice(gx_V s, int nres) {
    if (!s.u.p) return s;
    gx_V *a = gx_alloc_vals(s.l);
    for (uint32_t i = 0; i < s.l; i++) a[i] = gx_adapt(gx_vals(s)[i], nres);
    return gx_slice(a, s.l, s.l);
}

/* go f(args): starts a task running frame f. */
void gx_spawn(gx_V frame) {
    gx_Task *t = gx_new_task(gx_sched->next_id++, gx_frameof(frame));
    gx_ready(t);
}

void gx_spawn_call(gx_V f, int n, const gx_V *args) { gx_spawn(gx_sync_frame(gx_fnchk(f), n, args, 0)); }

static void (*main_init)(void);
static gx_V (*main_entry)(void);

static void main_body(void) {
    gx_Task *main = gx_new_task(0, NULL);
    gx_sched = gx_new_sched(main, false);
    gx_Handler h;
    if (GX_TRY(h)) gx_report_panic(gx_thrown);
    main_init();
    main->frame = gx_frameof(main_entry());
    GX_END(h);
    gx_ready(main);
    while (!main->done) run(next());
}

/* Runs the program entry as the first task until it returns. */
_Noreturn void gx_run_main(void (*init)(void), gx_V (*entry)(void)) {
    GC_INIT();
    main_init = init;
    main_entry = entry;
    gx_run_large(main_body);
    exit(0);
}

/* Requeues the running task: a pause primitive. */
void gx_yield_task(gx_Task *t) {
    gx_ready(t);
    gx_block(t);
}

static void await_step(gx_Task *t, gx_Frame *f) {
    if (f->pc == 0) {
        f->pc = 1;
        f->prim(t, f->prim_arg);
        return;
    }
    f->l[0] = gx_tuple(t->nrv, t->rv);
    gx_ret(t, f);
}

/* Runs one pause primitive in an isolated scheduler for a harness case and
 * stores its results in out. A blocked case raises with gx_blocked_signal
 * set; a panic raises the source panic. */
int gx_run_isolated(void (*prim)(gx_Task *, void *), void *arg, gx_V *out) {
    gx_Frame *h = gx_new_frame(1, await_step, NULL);
    h->prim = prim;
    h->prim_arg = arg;
    h->l[0] = gx_tuple(0, NULL);
    gx_Task *main = gx_new_task(0, h);
    gx_sched = gx_new_sched(main, true);
    gx_ready(main);
    while (!main->done) run(next());
    gx_V r = h->l[0];
    for (uint32_t i = 0; i < r.l; i++) out[i] = gx_at(r, (int)i);
    return (int)r.l;
}

void gx_reset_scheduler(void) {
    gx_sched = gx_new_sched(gx_new_task(0, NULL), true);
    gx_blocked_signal = 0;
}
