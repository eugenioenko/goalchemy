package rt;

/** lib.task.all: run each function as a task, in order, and resume the
 * caller when all have finished; a pause primitive. The functions arrive in
 * the resumable form (each call returns a frame). */
public final class LibTaskAll {
    private LibTaskAll() {}

    static final class AllChild extends TaskSpawn.Frame {
        final Fn f;
        final int[] n;
        final TaskSpawn.Task waiter;

        AllChild(Fn f, int[] n, TaskSpawn.Task waiter) {
            this.f = f;
            this.n = n;
            this.waiter = waiter;
        }

        public void step(TaskSpawn.Task t) {
            if (pc == 0) {
                pc = 1;
                if (f == null) throw Panics.nilDeref();
                TaskSpawn.call(t, (TaskSpawn.Frame) f.call());
                return;
            }
            if (--n[0] == 0) TaskSpawn.sched.ready(waiter);
            TaskSpawn.ret(t, this);
        }
    }

    public static void libTaskAll(TaskSpawn.Task t, Slice fns) {
        t.rv = new Object[0];
        if (fns.l == 0) return;
        int[] n = {fns.l};
        for (int i = 0; i < fns.l; i++) TaskSpawn.spawn(new AllChild((Fn) fns.get(i), n, t));
        TaskSpawn.sched.block(t);
    }
}
