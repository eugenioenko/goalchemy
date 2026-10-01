/* std.context.done. */
#include "gx.h"

gx_V gx_std_context_context_done(gx_V c) { return ((gx_Context *)gx_nilchk(c).u.p)->done; }
