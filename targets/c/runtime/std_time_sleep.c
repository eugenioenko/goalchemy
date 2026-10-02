/* std.time.sleep: blocks on the virtual clock. */
#include "gx.h"

void gx_std_time_sleep(gx_Task *t, gx_V d) {
    gx_owner_check(t->owner);
    gx_set_rv(t, 0, NULL);
    if (d.u.i <= 0) {
        gx_yield_task(t);
        return;
    }
    gx_add_timer(d.u.i, t, NULL, gx_nil());
    gx_block(t);
}
