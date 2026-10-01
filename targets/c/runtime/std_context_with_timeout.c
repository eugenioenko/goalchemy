/* std.context.with_timeout: returns (ctx, cancel); the deadline uses the virtual clock. */
#include "gx.h"

static void expire(gx_V c) { gx_cancel_ctx(c, gx_context_deadline_exceeded()); }

gx_V gx_std_context_with_timeout(gx_V parent, gx_V d) {
    gx_V c = gx_new_child(parent);
    if (((gx_Context *)c.u.p)->err.t == GX_NIL) gx_add_timer(d.u.i, NULL, expire, c);
    gx_V r[2] = {c, gx_func(-1, gx_cancel_code, 1, &c)};
    return gx_tuple(2, r);
}
