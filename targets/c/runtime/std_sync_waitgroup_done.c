/* std.sync.waitgroup.done. */
#include "gx.h"

void gx_std_sync_waitgroup_done(gx_V wg) { gx_std_sync_waitgroup_add(wg, gx_int(-1)); }
