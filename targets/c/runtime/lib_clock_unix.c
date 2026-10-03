#include "gx.h"
gx_V gx_lib_clock_unix(void) { struct timespec ts; if(clock_gettime(CLOCK_REALTIME,&ts))gx_host_fault("UTC clock failure"); return gx_int(ts.tv_sec); }
