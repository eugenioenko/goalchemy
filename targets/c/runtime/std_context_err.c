/* std.context.err and the context tree. */
#include "gx.h"

static gx_V canceled, deadline, bg;

static void init_values(void) {
    if (canceled.t != GX_NIL) return;
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

void gx_cancel_ctx(gx_V cv, gx_V err) {
    gx_Context *c = cv.u.p;
    if (c->err.t != GX_NIL) return;
    c->err = err;
    gx_chan_close(c->done);
    gx_V *ks = c->children;
    size_t n = c->n;
    c->children = NULL;
    c->n = c->cap = 0;
    for (size_t i = 0; i < n; i++) gx_cancel_ctx(ks[i], err);
}

gx_V gx_new_child(gx_V parent) {
    gx_Context *c = GC_MALLOC(sizeof(gx_Context));
    c->done = gx_make_chan(gx_int(0), gx_zero_nil);
    gx_V cv = gx_obj(c);
    gx_Context *p = parent.u.p;
    if (p->err.t != GX_NIL) {
        gx_cancel_ctx(cv, p->err);
    } else if (parent.u.p != gx_background().u.p) {
        if (p->n == p->cap) {
            size_t cap = p->cap ? p->cap * 2 : 4;
            gx_V *k = gx_alloc_vals(cap);
            if (p->n) memcpy(k, p->children, p->n * sizeof(gx_V));
            p->children = k;
            p->cap = cap;
        }
        p->children[p->n++] = cv;
    }
    return cv;
}

gx_V gx_cancel_code(gx_V *env, gx_V *args, int n) {
    (void)args;
    (void)n;
    gx_cancel_ctx(env[0], gx_context_canceled());
    return gx_nil();
}

gx_V gx_std_context_context_err(gx_V c) { return ((gx_Context *)gx_nilchk(c).u.p)->err; }
