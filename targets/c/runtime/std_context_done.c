/* std.context.done. */
#include "gx.h"

gx_V gx_std_context_context_done(gx_V c) {
    gx_V context = gx_nilchk(c);
    gx_observe_context(context);
    return ((gx_Context *)context.u.p)->done;
}
