namespace Rt;

/// Provider terminal settlement acknowledges release. Cancellation only requests a stop.
public static class Callback
{
    public delegate void Provider(Request request);
    public sealed class Registration{public readonly string Name;public readonly Provider Handler;public Registration(string name,Provider provider){Name=name;Handler=provider;}}
    public sealed class Request
    {
        readonly byte[] input;readonly HostToken token;
        bool settled,canceled,retained,starting=true,published;int stopHelpers;
        Action stop;Exception stopFault,submissionFault,pendingFault;byte[] pendingValue;string pendingRejection;
        internal Request(byte[] input,HostToken token){this.input=input;this.token=token;}
        public byte[] Input=>(byte[])input.Clone();public bool Canceled{get{lock(this)return canceled;}}
        public void Retain(){lock(this){if(!settled)retained=true;}}
        public void OnStop(Action hook)
        {
            Action invoke=null;
            lock(this){if(settled)return;retained=true;stop=hook??throw new ArgumentNullException(nameof(hook));if(canceled){stopHelpers++;invoke=stop;}}
            if(invoke!=null)InvokeStop(invoke);
        }
        void InvokeStop(Action hook)
        {
            Exception fault=null;try{hook();}catch(Exception e){fault=e;}
            finally{lock(this){if(fault!=null)stopFault=fault;stopHelpers--;PublishIfReady();}}
        }
        public void Cancel()
        {
            Action invoke=null;
            lock(this){if(canceled)return;canceled=true;if(!settled&&stop!=null){stopHelpers++;invoke=stop;}}
            if(invoke!=null)InvokeStop(invoke);
        }
        void PublishIfReady()
        {
            if(published||!settled||starting||stopHelpers!=0)return;published=true;
            var fault=stopFault??submissionFault??pendingFault;
            token.complete(new object[]{pendingRejection,pendingValue},fault==null?null:new HostFault("callback adapter failure"));
            pendingValue=null;pendingRejection=null;pendingFault=null;stop=null;token.acknowledgeCleanup();
        }
        void Finish(byte[] value,string reject,Exception fault)
        {
            lock(this){if(settled)return;settled=true;pendingValue=value==null?null:(byte[])value.Clone();pendingRejection=reject;pendingFault=fault;stop=null;PublishIfReady();}
        }
        internal void Started(){lock(this){starting=false;PublishIfReady();}}
        public void Resolve(byte[] value){if(value==null||value.Length>128<<10){Reject("provider: invalid response");return;}Finish(value,null,null);}
        public void Reject(string error)=>Finish(null,error??"provider: rejected",null);
        public void Fault(Exception error)=>Finish(null,null,error);
        internal void SubmissionFault(Exception error){bool pending;lock(this){submissionFault=error;pending=retained&&!settled;}if(pending)Cancel();else Finish(null,null,error);}
    }
    static Dictionary<string,Provider> current=new();
    public static Registration[] snapshot(Registration[] input)
    {if(input==null)return Array.Empty<Registration>();if(input.Length>1024)throw Library.invalid();var owned=new Registration[input.Length];var names=new HashSet<string>();for(int i=0;i<input.Length;i++){var r=input[i];if(r==null||r.Name==null||r.Name.Length==0||r.Name.Length>128||r.Handler==null||!names.Add(r.Name))throw Library.invalid();owned[i]=new Registration(Library.binaryString(r.Name),r.Handler);}return owned;}
    public static void install(Registration[] input){var map=new Dictionary<string,Provider>();foreach(var r in input)map.Add(r.Name,r.Handler);current=map;}
    public static Provider FromTask(Func<Request,System.Threading.Tasks.Task<byte[]>> provider)=>request=>
    {var task=provider(request)??throw new InvalidOperationException("null provider Task");request.Retain();_ = Settle(request,task);};
    static async System.Threading.Tasks.Task Settle(Request request,System.Threading.Tasks.Task<byte[]> task)
    {try{request.Resolve(await task.ConfigureAwait(false));}catch(Exception){request.Reject("provider: rejected");}}
    public static void request(GoTask task,GoContext context,string name,Slice input)
    {
        object[] zero={Slice.BYTE_NIL,null};if(context==null||context.owner!=null&&context.owner!=R.sched){task.rv=Native.failure(zero,"callback: invalid input");return;}
        R.observe(context);if(context.err!=null){task.rv=new object[]{Slice.BYTE_NIL,context.err};return;}if(!current.TryGetValue(name,out var provider)){task.rv=Native.failure(zero,"callback: provider unavailable");return;}
        byte[] owned;try{owned=Native.bytes(input);}catch(Native.Reject){task.rv=Native.failure(zero,"callback: invalid input");return;}if(owned.Length>128<<10){task.rv=Native.failure(zero,"callback: invalid input");return;}
        Request request=null;Action detach=()=>{};var token=R.sched.registerHost(task,()=>request.Cancel(),()=>detach(),()=>context.err==null?null:new object[]{Slice.BYTE_NIL,context.err},
            values=>values[0]!=null?Native.failure(zero,(string)values[0]):new object[]{Native.slice((byte[])values[1]),null});
        request=new Request(owned,token);detach=R.onCancel(context,request.Cancel);try{provider(request);}catch(Exception e){request.SubmissionFault(e);}finally{request.Started();}
    }
}
