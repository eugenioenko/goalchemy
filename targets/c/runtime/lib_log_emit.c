#include "gx.h"
void gx_lib_log_emit(gx_V level, gx_V unix_nano, gx_V message, gx_V attrs, gx_V text) { gx_log_emit(gx_i(level), gx_i(unix_nano), message, attrs, text); }
