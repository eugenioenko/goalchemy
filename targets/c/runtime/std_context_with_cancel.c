/* std.context.with_cancel: returns (ctx, cancel). */
#include "gx.h"

gx_V gx_std_context_with_cancel(gx_V parent) {
    gx_V c = gx_new_child(parent);
    gx_V r[2] = {c, gx_func(-1, gx_cancel_code, 1, &c)};
    return gx_tuple(2, r);
}
