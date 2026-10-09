#include "gx.h"
gx_V gx_lib_clock_unix_nano(void) { struct timespec ts; if(clock_gettime(CLOCK_REALTIME,&ts))gx_host_fault("UTC clock failure"); return gx_int((int64_t)ts.tv_sec*1000000000+ts.tv_nsec); }
