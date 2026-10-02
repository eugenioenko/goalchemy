/* std.context.err: owner-bound context tree and captured absolute deadlines. */
#include "gx.h"

static gx_V canceled, deadline, bg;
static void init_values(void) {
    if (canceled.t != GX_NIL)
        return;
    canceled = gx_std_errors_new(gx_cstr("context canceled"));
    deadline = gx_std_errors_new(gx_cstr("context deadline exceeded"));
    bg = gx_obj(GC_MALLOC(sizeof(gx_Context)));
}
gx_V gx_context_canceled(void) {
    init_values();
    return canceled;
}
gx_V gx_context_deadline_exceeded(void) {
    init_values();
    return deadline;
}
gx_V gx_background(void) {
    init_values();
    return bg;
}
static bool mutable_context(gx_Context *c) { return c && c->owner && gx_owner_current(c->owner) && !c->owner->retiring; }
static void detach_context(gx_Context *c) {
    gx_Context *p = c->parent;
    if (p) {
        size_t n = 0, old = p->n;
        for (size_t i = 0; i < old; i++)
            if (p->children[i].u.p != c)
                p->children[n++] = p->children[i];
        if (n < old)
            memset(p->children + n, 0, (old - n) * sizeof(gx_V));
        p->n = n;
    }
    c->parent = NULL;
    gx_remove_context_timer(c);
    gx_Context **link = &c->owner->contexts;
    while (*link && *link != c)
        link = &(*link)->next_owner;
    if (*link)
        *link = c->next_owner;
    c->next_owner = NULL;
}
void gx_cancel_ctx(gx_V cv, gx_V err) {
    gx_Context *c = cv.u.p;
    /* Supported captured source callbacks reject native and retired owners first. */
    if (!mutable_context(c) || c->err.t != GX_NIL)
        return;
    c->owner->context_cancel_depth++;
    c->err = err;
    gx_V *children = c->children;
    size_t n = c->n;
    c->children = NULL;
    c->n = c->cap = 0;
    /* This scanned detached vector remains rooted before close/recursive callbacks. */
    gx_chan_close(c->done);
    detach_context(c);
    for (size_t i = 0; i < n; i++) {
        ((gx_Context *)children[i].u.p)->parent = NULL;
        gx_cancel_ctx(children[i], err);
    }
    GC_reachable_here(children);
    if (--c->owner->context_cancel_depth == 0)
        gx_host_context_changed();
}
void gx_observe_context(gx_V context) {
    gx_Context *c = context.u.p;
    if (mutable_context(c) && c->err.t == GX_NIL && c->has_deadline && c->deadline <= gx_now())
        gx_cancel_ctx(context, gx_context_deadline_exceeded());
}
gx_V gx_new_child(gx_V parent) {
    gx_cur_task();
    gx_owner_check(gx_sched);
    if (parent.t == GX_NIL || !parent.u.p)
        gx_plain_panic("cannot create context from nil parent");
    gx_Context *p = parent.u.p;
    gx_observe_context(parent);
    if (p->owner && p->owner != gx_sched && p->owner->host)
        gx_host_fault("foreign source context");
    gx_Context *c = GC_MALLOC(sizeof(gx_Context));
    c->owner = gx_sched;
    c->next_owner = gx_sched->contexts;
    gx_sched->contexts = c;
    c->done = gx_make_chan(gx_int(0), gx_zero_nil);
    c->has_deadline = p->has_deadline;
    c->deadline = p->deadline;
    gx_V cv = gx_obj(c);
    if (p->err.t != GX_NIL)
        gx_cancel_ctx(cv, p->err);
    else if (p->has_deadline && gx_now() >= p->deadline)
        gx_cancel_ctx(cv, gx_context_deadline_exceeded());
    else if (parent.u.p != gx_background().u.p) {
        if (p->n == p->cap) {
            size_t cap = p->cap ? p->cap * 2 : 4;
            gx_V *children = gx_alloc_vals(cap);
            if (p->n)
                memcpy(children, p->children, p->n * sizeof(gx_V));
            p->children = children;
            p->cap = cap;
        }
        p->children[p->n++] = cv;
        c->parent = p;
    }
    return cv;
}
gx_V gx_cancel_code(gx_V *env, gx_V *args, int n) {
    (void)args;
    (void)n;
    gx_Context *c = env[0].u.p;
    if (!mutable_context(c))
        return gx_nil();
    gx_cancel_ctx(env[0], gx_context_canceled());
    return gx_nil();
}
gx_HostBoundary gx_host_boundary(gx_V context, int64_t timeout) {
    gx_owner_check(gx_sched);
    /* Capture request timeout at this source boundary, before native snapshot/submit. */
    gx_HostBoundary b = {.deadline_error = gx_context_deadline_exceeded()};
    if (context.t == GX_NIL || !context.u.p) {
        b.has_deadline = true;
        b.deadline = gx_now();
        b.deadline_error = gx_std_errors_new(gx_cstr("nil source context"));
        return b;
    }
    gx_Context *c = context.u.p;
    if (c->owner && c->owner != gx_sched) {
        b.has_deadline = true;
        b.deadline = gx_now();
        b.deadline_error = gx_std_errors_new(gx_cstr("foreign source context"));
        return b;
    }
    gx_observe_context(context);
    b.context = c;
    b.has_deadline = c->has_deadline;
    b.deadline = c->deadline;
    if (timeout > 0) {
        int64_t at = gx_deadline(gx_now(), timeout);
        if (!b.has_deadline || at < b.deadline) {
            b.has_deadline = true;
            b.deadline = at;
        }
    }
    return b;
}
gx_V gx_std_context_context_err(gx_V c) {
    gx_V context = gx_nilchk(c);
    gx_observe_context(context);
    return ((gx_Context *)context.u.p)->err;
}
