/* lib.task.all: runs each function in its own task and waits for all. */
#include "gx.h"

static void all_child_step(gx_Task *t, gx_Frame *f) {
    if (f->pc == 0) {
        f->pc = 1;
        gx_call(t, gx_callv(f->l[0], 0, NULL));
        return;
    }
    gx_std_sync_waitgroup_done(f->l[1]);
    gx_ret(t, f);
}

void gx_lib_task_all(gx_Task *t, gx_V fns) {
    gx_set_rv(t, 0, NULL);
    if (fns.l == 0) return;
    gx_V wg = gx_new_waitgroup();
    gx_std_sync_waitgroup_add(wg, gx_int(fns.l));
    for (uint32_t i = 0; i < fns.l; i++) {
        gx_Frame *c = gx_new_frame(2, all_child_step, NULL);
        c->l[0] = gx_vals(fns)[i];
        c->l[1] = wg;
        gx_spawn(gx_vframe(c));
    }
    gx_std_sync_waitgroup_wait(t, wg);
}
