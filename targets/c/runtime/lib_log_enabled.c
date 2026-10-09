#include "gx.h"
gx_V gx_lib_log_enabled(gx_V level) { return gx_bool(gx_log_enabled(gx_i(level))); }
