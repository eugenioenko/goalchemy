/* core.integer.compare: three-way comparison. */
#include "gx.h"

gx_V gx_compare_int(gx_V a, gx_V b) { return gx_int(a.u.i < b.u.i ? -1 : a.u.i > b.u.i); }

gx_V gx_compare_uint(gx_V a, gx_V b) { return gx_int(gx_cmpu(a, b)); }
