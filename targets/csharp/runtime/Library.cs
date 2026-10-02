namespace Rt;

/// Serialized copied value calls; source construction and retirement stay on the acquired owner.
public static class Library
{
    public sealed class Failure : Exception
    {
        public readonly string Kind;
        readonly Dictionary<string,object> fields;
        public Failure(string kind, IDictionary<string,object> fields = null, Exception cause = null) : base("goalchemy library: " + kind, cause)
        { Kind = kind; this.fields = CopyFields(fields); }
        public Dictionary<string,object> Fields => CopyFields(fields);
        static Dictionary<string,object> CopyFields(IDictionary<string,object> input)
        { var result = new Dictionary<string,object>(); if (input != null) foreach (var p in input) result.Add(p.Key,R.snapshot(p.Value)); return result; }
    }
    public sealed class Options { public Callback.Registration[] Callbacks; public System.Threading.CancellationToken Cancellation; }
    public sealed class Operation<T>
    {
        internal readonly System.Threading.Tasks.TaskCompletionSource<T> result = new(System.Threading.Tasks.TaskCreationOptions.RunContinuationsAsynchronously);
        internal volatile bool stopped; internal Action wake = () => {}; internal IJob job;
        public System.Threading.Tasks.Task<T> Completion => result.Task;
        public bool Cancel()
        {
            bool queued = false; Action notify; IJob detached=null;
            lock (queue) { if (result.Task.IsCompleted || stopped) return false; stopped = true; if (job != null && queue.Remove(job)) { detached=job; job = null; queued = true; } notify = wake; }
            if (queued) { Exception failure=null;try{detached.ReleaseRegistration();}catch(Exception e){failure=e;}result.TrySetException(failure==null?new Failure("canceled"):new Failure("host_fault",null,failure)); }
            notify(); return true;
        }
    }
    internal interface IJob { void Run(); void ReleaseRegistration(); }
    static readonly List<IJob> queue = new(); static bool running;
    internal static Func<System.Threading.ThreadStart,System.Threading.Thread> ownerThreads=work=>new System.Threading.Thread(work,64<<20){IsBackground=true,Name="goalchemy-library-owner"};
    sealed class Job<T> : IJob
    {
        readonly Operation<T> op; readonly Func<GoContext,Frame> entry; readonly Action reset; readonly Func<object[],T> capture; readonly Callback.Registration[] callbacks;
        internal System.Threading.CancellationTokenRegistration cancellation;
        internal Job(Operation<T> op,Func<GoContext,Frame> entry,Action reset,Func<object[],T> capture,Callback.Registration[] callbacks)
        { this.op=op;this.entry=entry;this.reset=reset;this.capture=capture;this.callbacks=callbacks; }
        public void ReleaseRegistration()=>cancellation.Dispose();
        internal void Register(System.Threading.CancellationToken token) { cancellation=token.Register(() => op.Cancel()); }
        void Retire()
        { Exception fault=null; try {Native.retire();} catch(Exception e){fault=e;} try {Callback.install(Array.Empty<Callback.Registration>());}catch(Exception e){fault=e;} try{reset();}catch(Exception e){fault=e;} if(fault!=null)throw new Failure("host_fault",null,fault); }
        public void Run()
        {
            T value=default; Exception failure=null; GoContext ctx=null; bool owns=false;
            try
            {
                if(op.stopped)throw new Failure("canceled");
                value=R.driveLibrary(() => { owns=true;reset();Native.install();Callback.install(callbacks);ctx=(GoContext)R.stdContextWithCancel(R.BACKGROUND)[0];return entry(ctx); },
                    rv => { if(op.stopped)throw new Failure("canceled");return capture(rv); },
                    () => { if(op.stopped && ctx!=null) { Native.requestStop();R.cancel(ctx,R.CONTEXT_CANCELED); } },
                    w => {lock(queue)op.wake=w;}, () => {if(owns)Retire();});
            }
            catch(Exception e){failure=e is Failure ? e : new Failure("host_fault",null,e);}
            finally {ctx=null;lock(queue){op.wake=()=>{};op.job=null;}cancellation.Dispose();}
            // TCS uses RunContinuationsAsynchronously: user code cannot block this owner or execute under its queue monitor.
            if(failure!=null)op.result.TrySetException(failure);else op.result.TrySetResult(value);
        }
    }
    public static Operation<T> failed<T>(Failure error){var op=new Operation<T>();op.result.SetException(error);return op;}
    public static Operation<T> submit<T>(Options options,Func<GoContext,Frame> entry,Action reset,Func<object[],T> capture)
    {
        Job<T> job=null;
        try
        {
            var callbacks=Callback.snapshot(options?.Callbacks); var op=new Operation<T>();job=new Job<T>(op,entry,reset,capture,callbacks);op.job=job;
            job.Register(options?.Cancellation ?? default);
            lock(queue)
            {
                queue.Add(job);if(!running){running=true;try{var owner=ownerThreads(Drain);owner.Start();}
                    catch(Exception e){queue.Remove(job);op.job=null;running=false;throw new Failure("host_fault",null,e);}}
            }
            return op;
        }
        catch(Failure e){try{job?.ReleaseRegistration();}catch(Exception cleanup){return failed<T>(new Failure("host_fault",null,cleanup));}return failed<T>(e);}
        catch(Exception e){try{job?.ReleaseRegistration();}catch(Exception cleanup){e=cleanup;}return failed<T>(new Failure("host_fault",null,e));}
    }
    static void Drain(){for(;;){IJob job;lock(queue){if(queue.Count==0){running=false;return;}job=queue[0];queue.RemoveAt(0);}job.Run();}}
    sealed class Sequence : Frame
    {
        readonly Frame init;readonly Func<Frame> call;object[] res=Array.Empty<object>();
        internal Sequence(Frame init,Func<Frame> call){this.init=init;this.call=call;}
        public override void step(GoTask t){if(pc==0){pc=1;R.call(t,init);return;}if(pc==1){pc=2;R.call(t,call());return;}res=t.rv;R.ret(t,this);}
        public override object[] results()=>res;
    }
    public static Frame sequence(Frame init,Func<Frame> call)=>new Sequence(init,call);
    public sealed class CopyContext
    {
        readonly HashSet<object> path=new(System.Collections.Generic.ReferenceEqualityComparer.Instance);int nodes;
        public void enter(object value){if(value!=null && (++nodes>1000000||path.Count>=128||!path.Add(value)))throw invalid();}
        public void leave(object value){if(value!=null)path.Remove(value);}
    }
    public static Failure invalid()=>new("invalid_argument");
    public static void length(int n){if(n>64<<20)throw invalid();}
    public static byte[] bytes(byte[] value){if(value==null)return null;length(value.Length);return (byte[])value.Clone();}
    public static string binaryString(string value){value??="";length(value.Length);foreach(char c in value)if(c>255)throw invalid();return value;}
    public static long integer(long value,int bits,bool signed){if(bits<64){long low=signed?-(1L<<(bits-1)):0,high=(1L<<(signed?bits-1:bits))-1;if(value<low||value>high)throw invalid();}return value;}
}
