namespace Rt;

/// Actual CLR runtime tests; HTTP adapters stay in copied integration output.
public static class HostOperationsTest
{
    static void check(bool value, string message) { if (!value) throw new Exception("ASSERT: " + message); }
    static void awaitGate(System.Threading.ManualResetEventSlim gate) => check(gate.Wait(10000), "gate timed out");
    static Frame idle() => R.sync(() => Array.Empty<object>());
    static Scheduler owner() { R.resetScheduler(); R.sched.hostMode=true; return R.sched; }
    static GoTask task(Scheduler s, int id) => new GoTask(id,idle()) { owner=s };
    static void wireAndMailbox()
    {
        var s=owner(); var main=s.main; main.frame=idle();
        var first=s.registerHost(main,()=>{});check(main.blocked,"park before callback");
        byte[] bytes={0,255,128}; object[] nested={bytes,new List<object>{new Dictionary<string,object>{{"bytes",bytes}}}};
        var copy=(byte[])R.snapshot(bytes);bytes[0]=7;check(copy[0]==0,"copied input before submission");
        first.complete(nested);bytes[1]=1;first.complete(new object[]{"duplicate"});
        s.drainHost();check(s.runq.Count==0,"no apply before cleanup ACK");first.acknowledgeCleanup();s.drainHost();
        check(((byte[])main.rv[0])[1]==255,"copied unsigned byte result");
        check(((byte[])((System.Collections.Generic.IDictionary<string,object>)((System.Collections.Generic.IList<object>)main.rv[1])[0])["bytes"])[2]==128,"nested wire copy");
        main.rv=Array.Empty<object>();s.runq.Clear();
        var a=task(s,1);var b=task(s,2);var ta=s.registerHost(a,()=>{});var tb=s.registerHost(b,()=>{});
        tb.complete(new object[]{2L});ta.complete(new object[]{1L});tb.acknowledgeCleanup();ta.acknowledgeCleanup();s.drainHost();
        check(s.runq.Dequeue()==b&&s.runq.Dequeue()==a,"out-of-order publication FIFO");
        // .NET Dictionary reuses removed slots: publication sequence must be explicit.
        var holeA=task(s,3);var holeB=task(s,4);var holeC=task(s,5);
        var ha=s.registerHost(holeA,()=>{});var hb=s.registerHost(holeB,()=>{});
        ha.complete(new object[]{3L});hb.complete(new object[]{4L});ha.acknowledgeCleanup();s.drainHost();check(s.runq.Dequeue()==holeA,"ACK hole prefix");
        var hc=s.registerHost(holeC,()=>{});hc.complete(new object[]{5L});hb.acknowledgeCleanup();hc.acknowledgeCleanup();s.drainHost();
        check(s.runq.Dequeue()==holeB&&s.runq.Dequeue()==holeC,"publication FIFO survives dictionary slot reuse and ACK hole");
        for(int i=0;i<10000;i++){first.complete(new object[]{i});first.acknowledgeCleanup();}
        check(s.mail.records.Count==0&&s.mail.cleaned.Count==0&&s.mail.live.Count==0,"bounded stale record/ACK bookkeeping");
        try { R.snapshot(new object[]{idle()}); throw new Exception("source frame accepted"); } catch(HostFault){}
        var cyclic=new object[1];cyclic[0]=cyclic;try{R.snapshot(cyclic);throw new Exception("cycle accepted");}catch(HostFault){}
        s.shutdown();var other=owner();first.complete(new object[]{3L});first.acknowledgeCleanup();other.drainHost();
        check(other.runq.Count==0,"retired generation ignored");
        // A token with this generation but wrong task ID cannot apply a record.
        var actual=other.registerHost(other.main,()=>{});var foreign=new HostToken(other.mail,actual.operation,999);
        foreign.complete(new object[]{3L});foreign.acknowledgeCleanup();other.drainHost();check(other.runq.Count==0&&other.operations.Count==1,"foreign task record ignored without poisoning operation");actual.complete(new object[]{4L});actual.acknowledgeCleanup();other.drainHost();check((long)other.main.rv[0]==4,"real token still applicable");other.shutdown();
    }
    static void precedence()
    {
        for(int mode=0;mode<3;mode++)
        {
            var s=owner();var t=s.main;t.frame=idle();
            var context=(GoContext)R.stdContextWithCancel(R.BACKGROUND)[0];int nativeCancel=0;
            var detach=R.onCancel(context,()=>nativeCancel++);
            var token=s.registerHost(t,()=>{},detach,()=>context.err==null?null:new object[]{null,context.err});
            token.complete(new object[]{"success"},mode==2?new HostFault("adapter fault"):null);token.acknowledgeCleanup();
            if(mode!=1)R.cancel(context,R.CONTEXT_CANCELED);
            try
            {
                s.drainHost();check(mode!=2,"fault must beat cancellation");
                if(mode==0)check(t.rv[1]==R.CONTEXT_CANCELED,"cancellation before apply wins");
                else {R.cancel(context,R.CONTEXT_CANCELED);check((string)t.rv[0]=="success","committed success final");}
            }
            catch(HostFault e){check(mode==2&&e.Message=="adapter fault","fault precedence category");}
            check(context.hooks.Count==0,"hook detached");s.shutdown();
        }
        var d=owner();var tok=d.registerHost(d.main,()=>{},decode:rv=>throw Panics.nilDeref());
        tok.complete(Array.Empty<object>());tok.acknowledgeCleanup();
        try{d.drainHost();throw new Exception("decode missing");}catch(HostFault e){check(e.Message.Contains("driver decode"),"decode fault outside source recover");}finally{d.shutdown();}
    }
    static void clocksAndContexts()
    {
        R.resetScheduler();var s=R.sched;
        var parent=(GoContext)R.stdContextWithTimeout(R.BACKGROUND,10)[0];s.clock=10;
        var child=(GoContext)R.stdContextWithTimeout(parent,100)[0];
        check(child.err==R.CONTEXT_DEADLINE_EXCEEDED&&parent.err==child.err,"expired parent immediately observed");
        check(s.timers.Count==0&&s.disposers.Count==0,"expired parent/child pruning");
        var zero=(GoContext)R.stdContextWithTimeout(R.BACKGROUND,0)[0];check(zero.err==R.CONTEXT_DEADLINE_EXCEEDED,"nonpositive timeout");
        s.clock=long.MaxValue-3;var saturated=(GoContext)R.stdContextWithTimeout(R.BACKGROUND,99)[0];check(saturated.deadline==long.MaxValue,"duration saturates");
        R.cancel(saturated,R.CONTEXT_CANCELED);s.shutdown();
        long nativeNow=700;var main=new GoTask(0,idle());s=new Scheduler(main,()=>nativeNow,1000000000);R.sched=s;s.hostMode=true;
        var order=new List<int>();s.addTimerAt(20,null,()=>order.Add(3));s.addTimerAt(10,null,()=>order.Add(1));s.addTimerAt(10,null,()=>order.Add(2));
        s.ready(main);nativeNow=710;check(s.nextHost()==main&&order.Count==2&&order[0]==1&&order[1]==2,"due real timers with runnable task ordered by deadline and sequence");
        nativeNow=709;check(s.now()==10,"clock cannot regress");
        var ctx=(GoContext)R.stdContextWithTimeout(R.BACKGROUND,20)[0];check(ctx.deadline==30&&s.timers.Exists(t=>t.at==30),"absolute deadline");
        nativeNow=725;int canceled=0;var request=new HostBoundary(R.BACKGROUND,5,()=>canceled++);
        nativeNow=731;check(request.error()==R.CONTEXT_DEADLINE_EXCEEDED&&R.BACKGROUND.err==null&&canceled==1,"request-only timeout anchored before delayed submission");request.Dispose();s.shutdown();
        long drift=900;s=new Scheduler(new GoTask(0,idle()),()=>drift++,1000000000);R.sched=s;s.hostMode=true;
        var anchored=(GoContext)R.stdContextWithTimeout(R.BACKGROUND,10)[0];
        check(anchored.deadline==11&&s.timers[0].at==anchored.deadline,"no resampling drift in context timer");s.shutdown();
        long ticks=0;s=new Scheduler(new GoTask(0,idle()),()=>ticks,3);R.sched=s;s.hostMode=true;ticks=2;
        check(s.now()==666666666,"single rounded tick conversion");ticks=long.MaxValue;check(s.now()==long.MaxValue,"tick conversion saturates before narrowing");s.shutdown();
        for(int i=0;i<300;i++)
        {
            R.resetScheduler();s=R.sched;var root=(GoContext)R.stdContextWithCancel(R.BACKGROUND)[0];
            for(int n=0;n<5;n++)
            {
                var c=(GoContext)R.stdContextWithTimeout(root,long.MaxValue)[0];var remove=R.onCancel(c,()=>{});remove();R.cancel(c,R.CONTEXT_CANCELED);
            }
            check(root.children.Count==0&&s.timers.Count==0,"repeated child/timer/hook pruning");R.cancel(root,R.CONTEXT_CANCELED);check(s.disposers.Count==0,"disposer pruning");s.shutdown();
        }
        R.resetScheduler();try{R.stdContextContextErr(null);throw new Exception("nil accepted");}catch(GoPanic){}
        var foreignCtx=(GoContext)R.stdContextWithCancel(R.BACKGROUND)[0];R.resetScheduler();
        try{R.observe(foreignCtx);throw new Exception("foreign accepted");}catch(HostFault){}
    }
    static void blockedRoots()
    {
        foreach(string mode in new[]{"send","recv","select","mutex","waitgroup","nil-send","nil-recv","empty-select","sleep"})
        {
            var s=owner();var t=s.main;t.frame=idle();var ch=R.makeChan(0,()=>null);var second=R.makeChan(0,()=>null);
            var mutex=new GoMutex{locked=true};var wg=new WaitGroup{n=1};object payload=new byte[1048576];Waiter stale=null;
            switch(mode)
            {
                case "send":R.chanSend(t,ch,payload);check(ch.sendq.Count==1,"send populated");stale=ch.sendq[0];break;
                case "recv":R.chanRecv(t,ch);check(ch.recvq.Count==1,"recv populated");stale=ch.recvq[0];break;
                case "select":R.select(t,false,R.scase(ch,true,payload),R.scase(second,false,null));check(ch.sendq.Count==1&&second.recvq.Count==1,"select populated");stale=ch.sendq[0];break;
                case "mutex":R.stdSyncMutexLock(t,mutex);check(mutex.waiters.Count==1,"mutex populated");break;
                case "waitgroup":R.stdSyncWaitgroupWait(t,wg);check(wg.waiters.Count==1,"waitgroup populated");break;
                case "nil-send":R.chanSend(t,null,payload);break;
                case "nil-recv":R.chanRecv(t,null);break;
                case "empty-select":R.select(t,false);break;
                case "sleep":R.stdTimeSleep(t,long.MaxValue);check(s.timers.Count==1,"sleep populated");break;
            }
            check(t.blocked,"blocked before retirement: "+mode);s.shutdown();
            check(ch.sendq.Count==0&&ch.recvq.Count==0&&second.recvq.Count==0&&mutex.waiters.Count==0&&wg.waiters.Count==0,mode+" external roots pruned");
            check(t.frame==null&&t.rv.Length==0&&t.cleanup==null&&t.curPanic==null&&s.tasks.Count==0,mode+" task roots pruned");
            R.resetScheduler();if(stale!=null){check(stale.val==null&&stale.task==null,"retained waiter cleared");stale.recvDone(99L,true);stale.sendDone(true);}
            R.chanClose(ch);R.stdSyncMutexUnlock(mutex);R.stdSyncWaitgroupAdd(wg,-1);
            check(R.sched.runq.Count==0&&t.rv.Length==0&&t.resumePanic==null,"stale callback rejected");
        }
        var buffered=R.makeChan(2,()=>null);R.runIsolated(t=>{R.chanSend(t,buffered,null);R.chanSend(t,buffered,7L);});
        var result=R.runIsolated(t=>R.chanRecv(t,buffered));check(result[0]==null&&(bool)result[1],"nil buffer FIFO");
        result=R.runIsolated(t=>R.chanRecv(t,buffered));check((long)result[0]==7,"after nil FIFO");
    }
    static void cleanupFaults()
    {
        var s=owner();int cleanup=0;var a=s.main;var b=task(s,1);
        var ta=s.registerHost(a,()=>throw new Exception("cancel fault"),()=>{cleanup++;throw new Exception("cleanup fault");});
        var tb=s.registerHost(b,()=>{},()=>cleanup++);ta.acknowledgeCleanup();tb.acknowledgeCleanup();
        s.disposers.Add(()=>{cleanup++;throw new Exception("dispose fault");});s.disposers.Add(()=>cleanup++);
        a.cleanup=()=>{cleanup++;throw new Exception("task fault");};b.cleanup=()=>cleanup++;
        try{s.shutdown();throw new Exception("missing cleanup fault");}catch(HostFault){}
        check(cleanup==6&&s.tasks.Count==0&&s.disposers.Count==0&&s.operations.Count==0&&s.mail.cleaned.Count==0,"exhaustive cleanup after faults");
        R.resetScheduler();var c=(GoContext)R.stdContextWithCancel(R.BACKGROUND)[0];int hooks=0;
        R.onCancel(c,()=>throw new Exception("hook fault"));R.onCancel(c,()=>hooks++);
        try{R.cancel(c,R.CONTEXT_CANCELED);throw new Exception("missing hook fault");}catch(HostFault){}
        check(hooks==1&&c.hooks.Count==0&&c.children.Count==0,"exhaustive cancel hooks");
    }
    sealed class AsyncFrame : Frame
    {
        readonly Action<GoTask> start; readonly Action<GoTask> resume;
        public AsyncFrame(Action<GoTask> start, Action<GoTask> resume) { this.start=start;this.resume=resume; }
        public override void step(GoTask t)
        {
            if(pc++==0){start(t);return;}resume(t);R.ret(t,this);
        }
    }
    static void waitRetirement(Scheduler s)
    {
        check(System.Threading.SpinWait.SpinUntil(()=>{lock(s.mail)return s.closed&&s.mail.retiring&&s.mail.waiting;},10000),"owner reached real cleanup ACK wait");
    }
    static void realDriveAndOverlap()
    {
        using var started=new System.Threading.ManualResetEventSlim();using var release=new System.Threading.ManualResetEventSlim();
        int progress=0,resumed=0;
        var done=R.runMainHost(new AsyncFrame(t=>
        {
            R.spawn(R.sync(()=>{System.Threading.Interlocked.Increment(ref progress);return Array.Empty<object>();}));
            var token=R.sched.registerHost(t,release.Set);
            R.sched.launchHost(token,()=>System.Threading.Tasks.Task.Run(()=>{started.Set();awaitGate(release);return new object[]{77L};}));
        },t=>{check((long)t.rv[0]==77,"real native result");resumed++;}));
        awaitGate(started);
        try{R.runMainHost(idle());throw new Exception("overlap accepted");}catch(HostFault){}
        try{R.resetScheduler();throw new Exception("reset overlap accepted");}catch(HostFault){}
        try{R.runIsolated(t=>{});throw new Exception("harness overlap accepted");}catch(HostFault){}
        check(System.Threading.SpinWait.SpinUntil(()=>System.Threading.Volatile.Read(ref progress)==1,10000),"source progressed with native work pending");
        check(!done.IsCompleted,"pending prevents false deadlock");release.Set();done.GetAwaiter().GetResult();check(resumed==1,"native resume exactly once");
        R.runMainHost(new AsyncFrame(t=>
        {
            var token=R.sched.registerHost(t,()=>{});R.sched.launchHost(token,()=>System.Threading.Tasks.Task.FromResult(new object[]{88L}));
        },t=>check((long)t.rv[0]==88,"synchronous Task completion"))).GetAwaiter().GetResult();
        try{R.runMainHost(new AsyncFrame(t=>{var token=R.sched.registerHost(t,()=>{});R.sched.launchHost(token,()=>System.Threading.Tasks.Task.FromException<object[]>(new Exception("unexpected rejection")));},t=>throw new Exception("source resumed after host fault"))).GetAwaiter().GetResult();throw new Exception("fault missing");}
        catch(HostFault e){check(e.Message.Contains("unexpected Task exception"),"Task exception bypasses source panic");}
        check(Program.panicState()!=null&&Program.panicState().curPanic==null&&Program.panicState().deferTarget==null,"usable empty panic binding after retirement");
    }
    sealed class Progress : Frame
    {
        internal bool started;internal GoContext context;internal int ticks;
        public override void step(GoTask t)
        {
            started=true;
            if(context!=null&&R.stdContextContextErr(context)!=null){R.ret(t,this);return;}
            ticks++;R.yieldTask(t);
        }
    }
    sealed class RealDeadline : Frame
    {
        readonly Progress progress=new();GoContext parent,child;long began;
        public override void step(GoTask t)
        {
            if(pc==0){pc=1;R.spawn(progress);R.yieldTask(t);return;}
            if(pc==1)
            {
                check(progress.started,"progress barrier before measured deadline");pc=2;
                began=System.Diagnostics.Stopwatch.GetTimestamp();
                parent=(GoContext)R.stdContextWithTimeout(R.BACKGROUND,20000000)[0];
                child=(GoContext)R.stdContextWithTimeout(parent,100000000)[0];progress.context=parent;
                R.chanRecv(t,R.stdContextContextDone(child));return;
            }
            long elapsed=System.Diagnostics.Stopwatch.GetTimestamp()-began;
            check((UInt128)(ulong)elapsed*1000000000UL/(ulong)System.Diagnostics.Stopwatch.Frequency>=20000000,"real deadline never virtual-fast-forwards");
            check(progress.ticks>0&&parent.err==R.CONTEXT_DEADLINE_EXCEEDED&&child.err==parent.err,"real timer dispatched despite runnable source");R.ret(t,this);
        }
    }
    static void realDeadline() => R.runMainHost(new RealDeadline()).GetAwaiter().GetResult();
    sealed class Lease : IDisposable
    {
        public int released; public byte[] input;
        public Lease(byte[] input){this.input=input;}
        public void Dispose(){check(input[1]==255,"retained copied input");input=null;released++;}
    }
    sealed class KeyLease : IDisposable
    {
        public bool acquired=true;public int released;
        public void Dispose(){check(acquired,"key released exactly once");acquired=false;released++;}
    }
    static void retirementGates()
    {
        foreach(string mode in new[]{"return","step-fault","adapter-fault"})
        {
            using var requested=new System.Threading.ManualResetEventSlim();using var ack=new System.Threading.ManualResetEventSlim();
            Scheduler captured=null;HostToken token=null;Lease lease=null;KeyLease key=null;int releases=0;var canceledTask=new System.Threading.Tasks.TaskCompletionSource<object[]>();
            var entry=R.runMainHost(()=>new AsyncFrame(t=>
            {
                captured=R.sched;
                GoTask nativeTask=t;
                if(mode=="adapter-fault")
                {
                    // The failing operation owns no native lease; a separate
                    // pending background operation owns all retained resources.
                    var failing=captured.registerHost(t,()=>{},()=>releases++);
                    failing.complete(Array.Empty<object>(),new HostFault("applicable adapter fault"));failing.acknowledgeCleanup();
                    nativeTask=task(captured,1);
                }
                byte[] original={0,255,128};lease=new Lease((byte[])R.snapshot(original));original[1]=3;key=new KeyLease();
                try{R.snapshot(key);throw new Exception("opaque key snapshot accepted");}catch(HostFault){}
                token=captured.registerHost(nativeTask,()=>{canceledTask.TrySetCanceled();requested.Set();},()=>releases++);
                var nativeToken=token;
                _=System.Threading.Tasks.Task.Run(()=>{awaitGate(ack);lease.Dispose();key.Dispose();nativeToken.acknowledgeCleanup();});
                if(mode=="step-fault")throw new InvalidOperationException("source-step host fault");
                if(mode=="return")R.ret(t,t.frame);
            },t=>throw new Exception("must not resume")));
            awaitGate(requested);waitRetirement(captured);
            check(canceledTask.Task.IsCanceled&&!entry.IsCompleted&&lease.released==0&&lease.input[1]==255&&key.acquired&&key.released==0,"Task canceled does not release resources: "+mode);
            ack.Set();try{entry.GetAwaiter().GetResult();check(mode=="return","expected fault missing");}
            catch(HostFault e){check(mode=="adapter-fault"&&e.Message=="applicable adapter fault","adapter fault category");}
            catch(InvalidOperationException e){check(mode=="step-fault"&&e.Message=="source-step host fault","step fault category");}
            check(lease.released==1&&!key.acquired&&key.released==1&&releases==(mode=="adapter-fault"?2:1),"exact resource release before return");
            token.complete(new object[]{999L});token.acknowledgeCleanup();check(captured.mail.live.Count==0&&captured.mail.cleaned.Count==0,"late ACK bounded");
        }
    }
    static void resetGate()
    {
        using var requested=new System.Threading.ManualResetEventSlim();using var ack=new System.Threading.ManualResetEventSlim();
        Scheduler old=null;HostToken token=null;Exception fault=null;bool returned=false;
        var driver=new System.Threading.Thread(()=>
        {
            try
            {
                old=owner();token=old.registerHost(old.main,requested.Set);
                _=System.Threading.Tasks.Task.Run(()=>{awaitGate(ack);token.acknowledgeCleanup();});
                R.resetScheduler();returned=true;
            }
            catch(Exception e){fault=e;}
        });
        driver.Start();awaitGate(requested);waitRetirement(old);check(!returned,"reset awaits ACK");ack.Set();check(driver.Join(10000),"reset terminal");
        if(fault!=null)throw fault;check(returned&&old.tasks.Count==0&&old.operations.Count==0,"reset retires old owner");
        R.resetScheduler();token.complete(new object[]{123L});token.acknowledgeCleanup();check(R.sched.runq.Count==0&&old.mail.cleaned.Count==0,"reset stale token cannot revive");
    }
    sealed class StressMain : Frame
    {
        int seen;readonly Chan gate;readonly Func<int> resumed,cleaned;
        public StressMain(Chan gate,Func<int> resumed,Func<int> cleaned){this.gate=gate;this.resumed=resumed;this.cleaned=cleaned;}
        public override void step(GoTask t)
        {
            if(pc==1){seen++;pc=0;}
            if(seen==256){check(resumed()==256&&cleaned()==256,"all 256 resumes and cleanup before main return");R.ret(t,this);return;}
            pc=1;R.chanRecv(t,gate);
        }
    }
    static void concurrent256()
    {
        int resumed=0,cleaned=0;var gate=R.makeChan(256,()=>0L);
        R.runMainHost(()=>
        {
            for(int i=0;i<256;i++)
            {
                int expected=i;R.spawn(new AsyncFrame(worker=>
                {
                    var token=R.sched.registerHost(worker,()=>{},()=>cleaned++);
                    R.sched.launchHost(token,()=>System.Threading.Tasks.Task.Run(()=>new object[]{(long)expected,new byte[]{0,255,128}}));
                },worker=>{check((long)worker.rv[0]==expected&&((byte[])worker.rv[1])[1]==255,"every concurrent result");resumed++;R.chanSend(worker,gate,(long)expected);}));
            }
            return new StressMain(gate,()=>resumed,()=>cleaned);
        }).GetAwaiter().GetResult();check(resumed==256&&cleaned==256,"stress terminal counts");
    }
    static void fatal(string mode,string marker)
    {
        if(mode=="deadlock")
        {
            R.runMainHost(()=>{R.sched.disposers.Add(()=>File.WriteAllText(marker,mode+":clean"));return new AsyncFrame(t=>R.chanRecv(t,null),t=>{});}).GetAwaiter().GetResult();return;
        }
        using var cancel=new System.Threading.ManualResetEventSlim();Scheduler captured=null;
        R.runMainHost(()=>
        {
            captured=R.sched;
            R.spawn(new AsyncFrame(t=>
            {
                var token=R.sched.registerHost(t,cancel.Set);
                _=System.Threading.Tasks.Task.Run(()=>
                {
                    awaitGate(cancel);waitRetirement(captured);Console.WriteLine("owner waiting for cleanup ACK");Console.Out.Flush();
                    check(Console.ReadLine()=="release","fatal cleanup gate");File.WriteAllText(marker,mode+":clean");token.acknowledgeCleanup();
                });
            },t=>{}));
            return new AsyncFrame(t=>R.yieldTask(t),t=>
            {
                if(mode=="panic")throw Panics.plainPanic("source failure");
                if(mode=="mutex-fatal")R.stdSyncMutexUnlock(new GoMutex());
                throw new Exception("unknown fatal mode");
            });
        }).GetAwaiter().GetResult();
    }
    public static void Main(string[] args)
    {
        if(args.Length>0){fatal(args[0],args[1]);return;}
        wireAndMailbox();precedence();clocksAndContexts();blockedRoots();cleanupFaults();realDriveAndOverlap();realDeadline();retirementGates();resetGate();concurrent256();R.resetScheduler();
        Console.WriteLine("C# host operations passed");
    }
}
