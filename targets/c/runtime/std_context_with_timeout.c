/* std.context.with_timeout: captured absolute owner deadline, no resampling. */
#include "gx.h"
static void expire(gx_V c) { gx_cancel_ctx(c, gx_context_deadline_exceeded()); }
gx_V gx_std_context_with_timeout(gx_V parent, gx_V d) {
    int64_t at = gx_deadline(gx_now(), d.u.i);
    gx_V cv = gx_new_child(parent);
    gx_Context *c = cv.u.p;
    if (!c->has_deadline || at < c->deadline) {
        c->has_deadline = true;
        c->deadline = at;
    }
    if (c->err.t == GX_NIL) {
        if (c->deadline <= gx_now())
            gx_cancel_ctx(cv, gx_context_deadline_exceeded());
        else
            gx_add_timer_at(c->deadline, NULL, expire, cv);
    }
    gx_V result[2] = {cv, gx_func(-1, gx_cancel_code, 1, &cv)};
    return gx_tuple(2, result);
}
