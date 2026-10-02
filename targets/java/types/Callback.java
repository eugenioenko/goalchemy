package rt;

import java.util.Map;
import java.util.concurrent.CompletionStage;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.Function;

/** Explicit terminal settlement means all provider-owned work has released resources. */
public final class Callback {
    private Callback() {}
    @FunctionalInterface public interface Provider { void start(Request request) throws Exception; }
    public static final class Registration {
        public final String name;public final Provider provider;
        public Registration(String name,Provider provider){this.name=name;this.provider=provider;}
    }
    public static final class Request {
        private final byte[] input;private final TaskSpawn.HostToken token;
        private final AtomicBoolean settled=new AtomicBoolean();private final AtomicBoolean canceled=new AtomicBoolean();
        private volatile Runnable stop=()->{};
        private Throwable stopFault;private Throwable submissionFault;private boolean retained;
        Request(byte[] input,TaskSpawn.HostToken token){this.input=input;this.token=token;}
        public byte[] input(){return input.clone();}
        public boolean canceled(){return canceled.get();}
        /** Stop requests cancellation; it does not claim resource settlement. */
        public synchronized void retain(){retained=true;}
        public synchronized void onStop(Runnable hook){retained=true;if(hook==null)throw new IllegalArgumentException("null stop hook");stop=hook;if(canceled.get())invokeStop();}
        private synchronized void invokeStop(){try{stop.run();}catch(Throwable e){stopFault=e;}}
        public void cancel(){if(canceled.compareAndSet(false,true))invokeStop();}
        private boolean starting=true;private byte[] pendingValue;private String pendingRejection;private Throwable pendingFault;
        private void publish(){
            Throwable fault=pendingFault;
            if(submissionFault!=null)fault=submissionFault;if(stopFault!=null)fault=stopFault;
            token.complete(new Object[]{pendingRejection,pendingValue},fault==null?null:new TaskSpawn.HostFault("callback adapter: "+fault));
            pendingValue=null;pendingRejection=null;pendingFault=null;stop=()->{};
            token.acknowledgeCleanup();
        }
        private void finish(byte[] value,String rejection,Throwable fault){
            synchronized(this){
                if(!settled.compareAndSet(false,true))return;
                pendingValue=value==null?null:value.clone();pendingRejection=rejection;pendingFault=fault;
                if(!starting)publish();
            }
        }
        private synchronized void started(){starting=false;if(settled.get())publish();}
        public void resolve(byte[] value){if(value==null||value.length>128<<10){reject("provider: invalid response");return;}finish(value,null,null);}
        public void reject(String error){finish(null,error==null?"provider: rejected":error,null);}
        public void fault(Throwable fault){finish(null,null,fault);}
        private void submissionFault(Throwable fault){
            boolean pending;
            synchronized(this){submissionFault=fault;pending=retained&&!settled.get();}
            if(pending)cancel();else finish(null,null,fault);
        }
    }
    private static Map<String,Provider> current=Map.of();
    public static Registration[] snapshot(Registration[] input){
        if(input==null)return new Registration[0];if(input.length>1024)throw Library.invalid();
        Registration[] owned=new Registration[input.length];var names=new java.util.HashSet<String>();
        for(int i=0;i<input.length;i++){Registration r=input[i];if(r==null||r.name==null||r.name.isEmpty()||r.name.length()>128||r.provider==null||!names.add(r.name))throw Library.invalid();owned[i]=new Registration(Library.binaryString(r.name),r.provider);}
        return owned;
    }
    public static void install(Registration[] registrations){var map=new java.util.LinkedHashMap<String,Provider>();for(Registration r:registrations)map.put(r.name,r.provider);current=java.util.Collections.unmodifiableMap(map);}
    /** A failed stage is declared provider rejection; synchronous adapter throws are faults. */
    public static Provider fromStage(Function<Request,? extends CompletionStage<byte[]>> provider){
        return request->{CompletionStage<byte[]> stage=provider.apply(request);if(stage==null)throw new IllegalStateException("null provider stage");request.retain();try{stage.whenComplete((value,error)->{if(error!=null)request.reject("provider: rejected");else request.resolve(value);});}catch(Throwable e){request.submissionFault(e);}};
    }
    public static void request(TaskSpawn.Task task,StdContextErr.Context context,String name,Slice input){
        Object[] zero={Slice.BYTE_NIL,null};
        if(context==null||context.owner!=null&&context.owner!=TaskSpawn.sched){task.rv=Native.failure(zero,"callback: invalid input");return;}
        StdContextErr.observe(context);
        if(context.err!=null){task.rv=new Object[]{Slice.BYTE_NIL,context.err};return;}
        Provider provider=current.get(name);if(provider==null){task.rv=Native.failure(zero,"callback: provider unavailable");return;}
        byte[] owned;try{owned=Native.bytes(input);}catch(Native.Reject e){task.rv=Native.failure(zero,"callback: invalid input");return;}
        if(owned.length>128<<10){task.rv=Native.failure(zero,"callback: invalid input");return;}
        Request[] request={null};Runnable[] detach={()->{}};
        TaskSpawn.HostToken token=TaskSpawn.sched.registerHost(task,()->request[0].cancel(),()->detach[0].run(),
            ()->context.err==null?null:new Object[]{Slice.BYTE_NIL,context.err},
            values->values[0]!=null?Native.failure(zero,(String)values[0]):new Object[]{Native.slice((byte[])values[1]),null});
        request[0]=new Request(owned,token);detach[0]=StdContextErr.onCancel(context,request[0]::cancel);
        try{provider.start(request[0]);}catch(Throwable e){request[0].submissionFault(e);}finally{request[0].started();}
    }
}
