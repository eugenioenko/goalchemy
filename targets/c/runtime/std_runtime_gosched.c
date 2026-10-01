/* std.runtime.gosched: yields to the back of the run queue. */
#include "gx.h"

void gx_std_runtime_gosched(gx_Task *t) {
    gx_set_rv(t, 0, NULL);
    gx_yield_task(t);
}
