package rt;

import java.util.ArrayDeque;
import java.util.Arrays;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.CompletionStage;
import java.util.function.Function;
import java.util.function.Supplier;

/** Serialized value calls. Only the owner builds source objects or resets globals. */
public final class Library {
    private Library() {}
    public static final class Failure extends RuntimeException {
        public final String kind;
        private final Map<String,Object> fields;
        public Failure(String kind) { this(kind,Map.of()); }
        public Failure(String kind,Map<String,Object> fields) { this(kind,fields,null); }
        public Failure(String kind,Map<String,Object> fields,Throwable cause) {
            super("goalchemy library: "+kind,cause);this.kind=kind;
            this.fields=copyFields(fields);
        }
        public Map<String,Object> fields() { return copyFields(fields); }
        private static Map<String,Object> copyFields(Map<String,Object> input) {
            var result=new java.util.LinkedHashMap<String,Object>();
            for(var e:input.entrySet()) result.put(e.getKey(),copy(e.getValue()));
            return java.util.Collections.unmodifiableMap(result);
        }
        private static Object copy(Object v) {
            if(v instanceof byte[] a)return a.clone();
            if(v instanceof Object[] a) {var r=new Object[a.length];for(int i=0;i<a.length;i++)r[i]=copy(a[i]);return r;}
            if(v instanceof Map<?,?> m){var r=new java.util.LinkedHashMap<String,Object>();for(var e:m.entrySet())r.put((String)e.getKey(),copy(e.getValue()));return java.util.Collections.unmodifiableMap(r);}
            return v;
        }
    }
    public static final class Options {
        public Callback.Registration[] callbacks;
    }
    /** Cancellation requests a stop; completion remains pending until actual cleanup. */
    public static final class Operation<T> {
        private final CompletableFuture<T> result=new CompletableFuture<>();
        private volatile boolean stopped;
        private volatile Runnable wake=()->{};
        private Job<T> job;
        public CompletionStage<T> completion() { return result.minimalCompletionStage(); }
        public boolean cancel() {
            boolean queued=false;
            synchronized(queue) {
                if(result.isDone()||stopped)return false;stopped=true;
                if(job!=null&&queue.remove(job)){job=null;queued=true;}
            }
            // Completion continuations must never execute while holding the queue monitor.
            if(queued)result.completeExceptionally(new Failure("canceled"));
            wake.run();return true;
        }
    }
    private static final ArrayDeque<Job<?>> queue=new ArrayDeque<>();
    private static boolean running;
    // Isolated native-launch fixtures only; no public scheduling API.
    static volatile java.util.concurrent.ThreadFactory completionThreads=work->Thread.ofVirtual().name("goalchemy-library-completion").unstarted(work);
    static volatile java.util.concurrent.Executor completionFallback=work->java.util.concurrent.ForkJoinPool.commonPool().execute(work);
    private static final class Job<T> {
        final Operation<T> op; final Function<StdContextErr.Context,TaskSpawn.Frame> entry;
        final Runnable reset;final Function<Object[],T> capture;final Callback.Registration[] callbacks;
        Job(Operation<T> op,Function<StdContextErr.Context,TaskSpawn.Frame> entry,Runnable reset,
            Function<Object[],T> capture,Callback.Registration[] callbacks) {
            this.op=op;this.entry=entry;this.reset=reset;this.capture=capture;this.callbacks=callbacks;
        }
        private void retire(){
            Throwable failure=null;
            try{Native.retire();}catch(Throwable e){failure=e;}
            try{Callback.install(new Callback.Registration[0]);}catch(Throwable e){failure=e;}
            try{reset.run();}catch(Throwable e){failure=e;}
            if(failure!=null)throw new Failure("host_fault",Map.of(),failure);
        }
        boolean run() {
            T value=null;Throwable failure=null;
            final StdContextErr.Context[] ctx={null};final boolean[] owns={false};
            try {
                if(op.stopped)throw new Failure("canceled");
                value=TaskSpawn.driveLibrary(()-> {
                    owns[0]=true;reset.run();Native.install();Callback.install(callbacks);
                    ctx[0]=(StdContextErr.Context)StdContextWithCancel.stdContextWithCancel(StdContextErr.BACKGROUND)[0];
                    return entry.apply(ctx[0]);
                },rv-> {
                    if(op.stopped)throw new Failure("canceled");
                    return capture.apply(rv);
                },()-> {if(op.stopped&&ctx[0]!=null){ Native.requestStop();StdContextErr.cancel(ctx[0],StdContextErr.CONTEXT_CANCELED); }},w->op.wake=w,()-> {
                    if(owns[0])retire();
                });
            } catch(Throwable e) {failure=e instanceof Failure?e:new Failure("host_fault",Map.of(),e);}
            finally {
                ctx[0]=null;op.wake=()->{};op.job=null;
            }
            // Host continuations may submit and wait for another call. Publish copied
            // results after retirement on a native thread, keeping the owner queue free.
            T ownedValue=value;Throwable ownedFailure=failure;
            try {
                completionThreads.newThread(()->{
                    if(ownedFailure!=null)op.result.completeExceptionally(ownedFailure);else op.result.complete(ownedValue);
                }).start();
            }catch(Throwable launchFailure){
                Failure dispatchFault=new Failure("host_fault",Map.of(),launchFailure);
                try{completionFallback.execute(()->op.result.completeExceptionally(dispatchFault));}
                catch(Throwable fallbackFailure){
                    Failure terminal=new Failure("host_fault",Map.of(),fallbackFailure);terminal.addSuppressed(dispatchFault);
                    return emergencyPublication(op,terminal);
                }
            }
            return false;
        }
    }
    public static <T> Operation<T> failed(Failure error) {
        Operation<T> op=new Operation<>();op.result.completeExceptionally(error);return op;
    }
    public static <T> Operation<T> submit(Options options,Function<StdContextErr.Context,TaskSpawn.Frame> entry,
                                         Runnable reset,Function<Object[],T> capture) {
        try {
            Callback.Registration[] callbacks=Callback.snapshot(options==null?null:options.callbacks);
            Operation<T> op=new Operation<>();Job<T> job=new Job<>(op,entry,reset,capture,callbacks);op.job=job;
            synchronized(queue) {
                queue.add(job);
                if(!running) {
                    running=true;
                    try{Thread.ofPlatform().daemon().name("goalchemy-library-owner").stackSize(64L<<20).start(Library::drain);}
                    catch(Throwable launchFailure){queue.remove(job);op.job=null;running=false;throw new Failure("host_fault",Map.of(),launchFailure);}
                }
            }
            return op;
        } catch(Failure e) {return failed(e);}
    }
    private static void drain() {
        for(;;) {
            Job<?> job;
            synchronized(queue) {job=queue.poll();if(job==null){running=false;return;}}
            if(job.run())return;
        }
    }
    /** No independent publisher is available. Relinquish source-driver ownership
     * before synchronous emergency publication; arbitrary blocking host callback
     * graphs cannot be guaranteed parallel progress in this terminal path. */
    private static boolean emergencyPublication(Operation<?> current,Failure fault){
        Job<?>[] detached;
        synchronized(queue){detached=queue.toArray(Job<?>[]::new);queue.clear();running=false;for(Job<?> job:detached){job.op.job=null;job.op.wake=()->{};}}
        for(Job<?> job:detached)job.op.result.completeExceptionally(fault);
        current.result.completeExceptionally(fault);
        return true;
    }
    private static final class Sequence extends TaskSpawn.Frame {
        final TaskSpawn.Frame init;final Supplier<TaskSpawn.Frame> call;Object[] res=new Object[0];
        Sequence(TaskSpawn.Frame init,Supplier<TaskSpawn.Frame> call) {this.init=init;this.call=call;}
        public void step(TaskSpawn.Task t) {
            if(pc==0){pc=1;TaskSpawn.call(t,init);return;}
            if(pc==1){pc=2;TaskSpawn.call(t,call.get());return;}
            res=t.rv;TaskSpawn.ret(t,this);
        }
        public Object[] results(){return res;}
    }
    public static TaskSpawn.Frame sequence(TaskSpawn.Frame init,Supplier<TaskSpawn.Frame> call) {return new Sequence(init,call);}
    public static final class CopyContext {
        private final java.util.IdentityHashMap<Object,Boolean> path=new java.util.IdentityHashMap<>();
        private int nodes;
        public void enter(Object value){if(value==null)return;if(++nodes>1000000||path.size()>=128||path.put(value,true)!=null)throw invalid();}
        public void leave(Object value){if(value!=null)path.remove(value);}
    }
    public static Failure invalid(){return new Failure("invalid_argument");}
    public static void length(int n){if(n>64<<20)throw invalid();}
    public static byte[] bytes(byte[] v){if(v==null)return null;length(v.length);return v.clone();}
    public static String binaryString(String s){if(s==null)return "";length(s.length());for(int i=0;i<s.length();i++)if(s.charAt(i)>255)throw invalid();return s;}
    public static long integer(long value,int bits,boolean signed) {
        if(bits<64) {long lo=signed?-(1L<<(bits-1)):0,hi=(1L<<(signed?bits-1:bits))-1;if(value<lo||value>hi)throw invalid();}
        return value;
    }
}
