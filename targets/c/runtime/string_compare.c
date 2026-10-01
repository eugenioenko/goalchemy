/* core.string.compare: bytewise three-way comparison. */
#include "gx.h"

gx_V gx_scompare(gx_V a, gx_V b) { return gx_int(gx_scmp(a, b)); }
