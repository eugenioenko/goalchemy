/* core.chan.len. */
#include "gx.h"

gx_V gx_chan_len(gx_V ch) { return gx_int(ch.t == GX_NIL ? 0 : (int64_t)((gx_Chan *)ch.u.p)->len); }
