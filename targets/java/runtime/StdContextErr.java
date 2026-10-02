package rt;

import java.util.ArrayList;
import java.util.LinkedHashSet;
import java.util.Set;

/** Contexts and hooks belong to their creating driver; native callbacks publish
 * only to the host mailbox. Background has no owner and can cross entries. */
public final class StdContextErr {
    private StdContextErr() {}

    public static final class Context {
        final ChanMake.Chan done;
        Box err;
        final Set<Context> children = new LinkedHashSet<>();
        final Set<Runnable> hooks = new LinkedHashSet<>();
        TaskSpawn.Scheduler owner;
        Context parent;
        Long deadline;
        Runnable release, stopTimer;
        Context(ChanMake.Chan done) { this.done=done; }
    }
    public static final Box CONTEXT_CANCELED = StdErrorsNew.stdErrorsNew("context canceled");
    public static final Box CONTEXT_DEADLINE_EXCEEDED = StdErrorsNew.stdErrorsNew("context deadline exceeded");
    public static final Context BACKGROUND = new Context(null);

    public static void observe(Context c) {
        if(c==null) throw Panics.nilDeref();
        TaskSpawn.sched.assertDriver();
        if(c.owner!=null && c.owner!=TaskSpawn.sched) throw new TaskSpawn.HostFault("foreign context owner");
        if(c.err==null && c.deadline!=null && c.deadline<=TaskSpawn.sched.now()) cancel(c,CONTEXT_DEADLINE_EXCEEDED);
    }
    public static void cancel(Context c, Box err) {
        if(c==BACKGROUND) return;
        if(c.owner!=TaskSpawn.sched) throw new TaskSpawn.HostFault("foreign context owner");
        c.owner.assertDriver();
        if(c.err!=null) return;
        c.err=err;
        if(c.release!=null) { c.release.run(); c.release=null; }
        if(c.stopTimer!=null) { c.stopTimer.run(); c.stopTimer=null; }
        if(c.parent!=null) { c.parent.children.remove(c); c.parent=null; }
        ChanClose.chanClose(c.done);
        Throwable failure=null;
        for(Context child:new ArrayList<>(c.children)) try { cancel(child,err); } catch(Throwable e) { if(failure==null) failure=e; }
        c.children.clear();
        for(Runnable hook:new ArrayList<>(c.hooks)) try { hook.run(); } catch(Throwable e) { if(failure==null) failure=e; }
        c.hooks.clear();
        if(failure!=null) throw new TaskSpawn.HostFault("context cancel hook: " + failure);
    }
    static Context newChild(Context parent) {
        observe(parent);
        Context c=new Context(ChanMake.makeChan(0,()->null));
        TaskSpawn.Scheduler owner=TaskSpawn.sched; c.owner=owner;
        Runnable dispose=()-> {
            if(c.stopTimer!=null) c.stopTimer.run(); c.stopTimer=null;
            if(c.parent!=null) c.parent.children.remove(c); c.parent=null;
            c.children.clear(); c.hooks.clear(); c.release=null;
        };
        owner.disposers.add(dispose);
        c.release=()->owner.disposers.remove(dispose);
        c.deadline=parent.deadline;
        if(parent.err!=null) cancel(c,parent.err);
        else if(parent!=BACKGROUND) { parent.children.add(c); c.parent=parent; }
        return c;
    }
    public static Runnable onCancel(Context c,Runnable hook) {
        observe(c);
        if(c==BACKGROUND) return ()->{};
        if(c.err!=null) { hook.run(); return ()->{}; }
        c.hooks.add(hook);
        return ()->{ c.owner.assertDriver(); c.hooks.remove(hook); };
    }
    public static Box stdContextContextErr(Context c) { observe(c); return c.err; }

    /** Generic request-only timeout anchored BEFORE input copying/submission.
     * This is lifecycle plumbing, not an HTTP capability. Adapters must return
     * declared validation errors for nil/foreign contexts before using it. */
    public static final class Boundary {
        public final long deadline;
        private final TaskSpawn.Scheduler owner;
        private final Context context;
        private Box error;
        private Runnable stopTimer, detach, dispose;
        private final Runnable cancelNative;
        private boolean closed;
        public Boundary(Context context,long timeout,Runnable cancelNative) {
            observe(context);
            this.owner=TaskSpawn.sched; this.context=context; this.cancelNative=cancelNative;
            long requested=TaskSpawn.deadlineAfter(owner.now(),timeout);
            this.deadline=context.deadline==null ? requested : Math.min(requested,context.deadline);
            dispose=this::close; owner.disposers.add(dispose);
            detach=onCancel(context,()->expire(context.err));
            if(error==null && deadline<=owner.now()) expire(CONTEXT_DEADLINE_EXCEEDED);
            if(error==null) stopTimer=owner.addTimerAt(deadline,null,()->expire(CONTEXT_DEADLINE_EXCEEDED));
        }
        private void expire(Box err) {
            if(closed || error!=null) return;
            error=err;
            cancelNative.run();
        }
        public Box error() {
            owner.assertDriver(); observe(context);
            if(error==null && deadline<=owner.now()) expire(CONTEXT_DEADLINE_EXCEEDED);
            return error;
        }
        public void close() {
            owner.assertDriver(); if(closed) return; closed=true;
            if(stopTimer!=null) stopTimer.run(); stopTimer=null;
            if(detach!=null) detach.run(); detach=null;
            owner.disposers.remove(dispose); dispose=null;
        }
    }
}
