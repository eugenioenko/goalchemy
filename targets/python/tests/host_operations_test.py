"""Actual CPython generic host lifecycle. No production HTTP/crypto adapter."""
import concurrent.futures
import os
import sys
import threading
import time
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parents[2]))
from python.runtime import task_spawn as R
from python.runtime import std_context_err as C
from python.runtime.std_context_with_timeout import std_context_with_timeout
from python.runtime.std_context_with_cancel import std_context_with_cancel
from python.runtime.std_time_sleep import std_time_sleep
from python.runtime.chan_make import make_chan, Waiter, attach_waiter, dequeue, try_recv
from python.runtime.chan_send import chan_send
from python.runtime.chan_recv import chan_recv
from python.runtime.select import select
from python.runtime.std_sync_mutex_lock import Mutex, std_sync_mutex_lock
from python.runtime.std_sync_waitgroup_add import WaitGroup, std_sync_waitgroup_add
from python.runtime.std_sync_waitgroup_wait import std_sync_waitgroup_wait
from python.types.program import go_panic, STRING_TYPE, source_guard, source_guard_state
from python.types.iface import box
from python.types.panic import GoPanic


class Once(R.Frame):
    def __init__(self, fn):
        super().__init__()
        self.fn = fn
    def step(self, t):
        self.fn(t)
        R.ret(t, self)


def owner(fn):
    R.run_main_host(lambda: Once(fn))


def check_fault(fn, text):
    try:
        fn()
    except R.HostFault as e:
        assert text in str(e), str(e)
    else:
        raise AssertionError('missing host fault')


def identity_fifo_wire():
    saved = []
    def run(main):
        s = R.sched()
        tasks = [R.Task(i+1, None) for i in range(5)]
        cleaned = []
        tokens = [R.register_host(t, cleanup=lambda i=i: cleaned.append(i)) for i,t in enumerate(tasks)]
        saved.extend(tokens)
        wrong = R.HostToken(s.mailbox, tokens[0].operation, 999)
        assert not wrong.publish(fault='foreign poison') and not wrong.acknowledge_cleanup()
        for oid,tid in [(float(tokens[0].operation),tasks[0].id),(tokens[0].operation,float(tasks[0].id)),([],tasks[0].id)]:
            malformed=R.HostToken(s.mailbox,oid,tid)
            assert not malformed.publish(fault='malformed token') and not malformed.acknowledge_cleanup()
        original = [bytearray([0,255,128]), {'nested': bytearray([255])}]
        tokens[0].publish(original); original[0][1]=1
        tokens[1].publish([1]); tokens[0].acknowledge_cleanup()
        s.drain()
        assert tasks[0].rv == [b'\0\xff\x80', {'nested': b'\xff'}]
        assert cleaned == [0]
        # ACK hole remains while earlier dictionary slot is removed/reinserted.
        tokens[2].publish([2]); tokens[2].acknowledge_cleanup()
        tokens[1].acknowledge_cleanup(); s.drain()
        assert [t.id for t in s.runq] == [1,2,3]
        assert not tokens[0].publish(fault='late') and not tokens[0].acknowledge_cleanup()
        assert not tokens[2].publish([88])
        tokens[4].publish([4]);tokens[3].publish([3])
        tokens[3].acknowledge_cleanup();tokens[4].acknowledge_cleanup();s.drain()
        assert [t.id for t in s.runq] == [1,2,3,5,4]
        assert cleaned == [0,1,2,4,3]
        assert not s.mailbox.live and not s.mailbox.records and not s.mailbox.acks
        for bad in [main, lambda:None, object(), {1:'x'}]:
            check_fault(lambda bad=bad:R.snapshot(bad), 'wire')
        cycle=[];cycle.append(cycle)
        check_fault(lambda:R.snapshot(cycle),'cyclic')
        class Key: pass
        check_fault(lambda:R.snapshot(Key()),'wire')
    owner(run)
    for token in saved:
        assert not token.publish([999]) and not token.acknowledge_cleanup()
    owner(lambda t: None)


def cancellation_faults():
    def run(main):
        s=R.sched()
        c, cancel=std_context_with_cancel(C.BACKGROUND)
        t=R.Task(1,None)
        token=R.register_host(t,c,cancel_result=lambda err:[err])
        token.publish([b'success']);token.acknowledge_cleanup();cancel();s.drain()
        assert t.rv == [C.CONTEXT_CANCELED]
        c,cancel=std_context_with_cancel(C.BACKGROUND)
        t=R.Task(2,None);token=R.register_host(t,c)
        token.publish([b'final']);token.acknowledge_cleanup();s.drain();cancel()
        assert t.rv==[b'final']
    owner(run)
    def fault(main):
        c,cancel=std_context_with_cancel(C.BACKGROUND)
        token=R.register_host(main,c)
        token.publish(fault='native fault beats cancellation');token.acknowledge_cleanup();cancel()
        R.sched().drain()
    check_fault(lambda:owner(fault),'beats cancellation')
    def decode(main):
        token=R.register_host(main,decode=lambda _:(_ for _ in ()).throw(ValueError('decode failed')))
        token.publish([]);token.acknowledge_cleanup();R.sched().drain()
    check_fault(lambda:owner(decode),'decode failed')
    def native_exception(main):
        f=concurrent.futures.Future();f.set_exception(ValueError('unexpected worker'))
        R.launch_host(main,lambda _:f);R.sched().drain()
    check_fault(lambda:owner(native_exception),'unexpected worker')
    class Unprintable(Exception):
        def __str__(self):raise RuntimeError('broken fault formatter')
    def callback_exception(main):
        f=concurrent.futures.Future();f.set_exception(Unprintable())
        R.launch_host(main,lambda _:f);R.sched().drain()
    check_fault(lambda:owner(callback_exception),'unprintable native fault')
    owner(lambda t: check_fault(lambda:R.run_main_host(lambda:Once(lambda t:None)),'overlapping'))
    try:
        R.run_main_host(lambda:(_ for _ in ()).throw(ValueError('construction')))
    except ValueError: pass
    owner(lambda t:None)
    owner(lambda t: check_fault(lambda:R.reset_scheduler(),'overlapping'))
    # Constructor failure must release the shared reservation before re-entry.
    original=R._AwaitFrame
    def broken(fn):raise ValueError('harness construction')
    R._AwaitFrame=broken
    try:
        try:R.run_isolated(lambda t:None)
        except ValueError as e:assert str(e)=='harness construction'
        else:raise AssertionError('constructor fault missing')
    finally:R._AwaitFrame=original
    owner(lambda t:None)


def context_timers():
    def run(main):
        s=R.sched();native=[100];s.native_clock=lambda:native[0];s.epoch=0;s.clock=0
        c,cancel=std_context_with_timeout(C.BACKGROUND,20)
        assert c.deadline==120
        native[0]=105
        nested,nc=std_context_with_timeout(c,1000)
        assert nested.deadline==120 and nested.timer[0]==120
        request=C.host_boundary(C.BACKGROUND,10)
        assert request.deadline==115
        native[0]=115;s.dispatch()
        assert request.err is C.CONTEXT_DEADLINE_EXCEEDED and c.err is None
        native[0]=120;s.dispatch()
        assert c.err is nested.err is C.CONTEXT_DEADLINE_EXCEEDED
        expired,_=std_context_with_timeout(C.BACKGROUND,0)
        child,_=std_context_with_timeout(expired,1)
        assert child.err is C.CONTEXT_DEADLINE_EXCEEDED
        order=[]
        s.add_timer_at(130,None,lambda:order.append(1));s.add_timer_at(129,None,lambda:order.append(0))
        s.add_timer_at(130,None,lambda:order.append(2));native[0]=130;s.dispatch()
        assert order==[0,1,2]
        assert R.deadline(R.MAX_TIME-2,10)==R.MAX_TIME
        check_fault(lambda:R.deadline(1,R.MAX_TIME+1),'int64')
        for i in range(500):
            parent,pc=std_context_with_timeout(C.BACKGROUND,100)
            child,cc=std_context_with_cancel(parent)
            detach=child.add_hook(lambda:None);detach();cc();pc()
        assert not s.contexts and not s.timers
        try:C.check_context(None)
        except GoPanic:pass
        else:raise AssertionError('nil context must source panic')
        foreign=C.Context(None,R.Scheduler(R.Task(0,None)))
        check_fault(lambda:C.check_context(foreign),'foreign')
    owner(run)
    # Native wait ceiling must not overflow Condition.wait on enormous deadlines.
    observed=[];native_wait=threading.Event()
    class NativeCondition(threading.Condition):
        def wait(self, timeout=None):
            observed.append(timeout);native_wait.set()
            return super().wait(timeout)
    class Wake(R.Frame):
        def step(self,t):
            if self.pc==0:
                self.pc=1;s=R.sched();s.mailbox.condition=NativeCondition();s.add_timer(R.MAX_TIME,None,lambda:None)
                token=R.register_host(t)
                threading.Thread(target=lambda:(native_wait.wait(5),token.publish([7]),token.acknowledge_cleanup())).start()
            else:
                assert t.rv==[7];R.ret(t,self)
    R.run_main_host(Wake)
    assert observed and observed[0]==threading.TIMEOUT_MAX


def foreign_cancel_thread_checks():
    bg_removers=[]
    for i in range(8):
        owner(lambda t:bg_removers.append(C.BACKGROUND.add_hook(lambda:None)))
        assert not C.BACKGROUND.hooks
    for remove in bg_removers:
        check_fault(remove,'foreign')
    captured=[]
    def capture(t):
        c,cancel=std_context_with_cancel(C.BACKGROUND)
        captured.extend([c,cancel,c.add_hook(lambda:None)])
    owner(capture)
    R.reset_scheduler()
    check_fault(captured[1],'foreign')
    check_fault(captured[2],'foreign')
    check_fault(lambda:captured[0].add_hook(lambda:None),'foreign')
    def run(main):
        s=R.sched();failures=[]
        ch=make_chan(0);m=Mutex();wg=WaitGroup()
        funcs=[lambda:R.sched(),lambda:s.block(main),lambda:s.remove_timer([1,2,None,None]),
               lambda:s.drain(),lambda:chan_send(main,ch,b'x'),lambda:chan_recv(main,ch),
               lambda:select(main,[(ch,True,b'x')],False),lambda:std_sync_mutex_lock(main,m),
               lambda:std_sync_waitgroup_wait(main,wg),lambda:C.cancel_context(c,C.CONTEXT_CANCELED)]
        c,_=std_context_with_cancel(C.BACKGROUND)
        hook=lambda:None
        remover=c.add_hook(hook)
        funcs.append(remover)
        sender=R.Task(5,Once(lambda t:None));receiver=R.Task(6,Once(lambda t:None))
        selected=R.Task(7,Once(lambda t:None))
        send_channel=make_chan(0);recv_channel=make_chan(0);select_channel=make_chan(0)
        chan_send(sender,send_channel,bytearray([255]));chan_recv(receiver,recv_channel)
        select(selected,[(select_channel,True,bytearray([128])),(select_channel,False,None)],False)
        sw=send_channel.sendq[0];rw=recv_channel.recvq[0];tw=select_channel.sendq[0];tr=select_channel.recvq[0]
        fresh=Waiter(main,bytearray([1]))
        funcs.extend([lambda:sw.send_done(True),lambda:rw.recv_done(b'foreign',True),
                      lambda:tw.send_done(True),lambda:tr.recv_done(b'foreign',True),
                      sw.detach,lambda:dequeue(send_channel.sendq),lambda:try_recv(send_channel),
                      lambda:attach_waiter(main,ch.sendq,fresh)])
        def native():
            for f in funcs:
                try:f()
                except R.HostFault:failures.append(1)
        thread=threading.Thread(target=native);thread.start();thread.join(5)
        assert len(failures)==len(funcs) and not main.blocked and not ch.sendq and not ch.recvq
        assert not m.locked and not wg.waiters and c.err is None and hook in c.hooks
        assert sender.rv==receiver.rv==selected.rv==[] and not tw.sel.done
        assert sender.resume_panic is receiver.resume_panic is selected.resume_panic is None
        assert sw.task is sender and sw.val==bytearray([255])
        assert tr.task is selected and not ch.sendq
        assert len(send_channel.sendq)==len(recv_channel.recvq)==len(select_channel.sendq)==1
        remover();assert hook not in c.hooks
        done=R.Task(99,None);done.done=True
        check_fault(lambda:R.register_host(done),'task owner')
    owner(run)


def blocked_roots():
    for category in ['send','recv','select','mutex','waitgroup','nil-send','nil-recv','empty-select','sleep']:
        saved=[]
        def run(main):
            t=R.Task(1,Once(lambda t:None));s=R.sched();s.tasks.add(t);t.owner=s
            if category in ('send','recv','select'):
                ch=make_chan(0);saved.append(ch)
                if category=='send':chan_send(t,ch,bytearray(1024))
                elif category=='recv':chan_recv(t,ch)
                else:select(t,[(ch,True,bytearray(1024)),(ch,False,None)],False)
                assert len(ch.sendq)+len(ch.recvq)>0
                saved.extend(ch.sendq+ch.recvq)
            elif category=='mutex':
                m=Mutex();m.locked=True;std_sync_mutex_lock(t,m);assert m.waiters==[t];saved.append(m)
            elif category=='waitgroup':
                wg=WaitGroup();wg.n=1;std_sync_waitgroup_wait(t,wg);assert wg.waiters==[t];saved.append(wg)
            elif category=='nil-send':chan_send(t,None,bytearray(1024))
            elif category=='nil-recv':chan_recv(t,None)
            elif category=='empty-select':select(t,[],False)
            else:std_time_sleep(t,1000000000);assert s.timers
            saved.extend([s,t])
        owner(run)
        s,t=saved[-2:]
        assert not s.tasks and not s.timers and not s.runq
        assert t.frame is None and t.rv==[] and t.cleanup is None and t.done
        if category in ('send','recv','select'):
            ch=saved[0];assert not ch.sendq and not ch.recvq
            for w in saved[1:-2]:
                assert w.task is None and w.val is None
                w.recv_done(b'late',True);w.send_done(False)
        elif category in ('mutex','waitgroup'):assert not saved[0].waiters
        assert not s.runq


def cleanup_leases():
    for mode in ['return','exception','fault','reset']:
        started=threading.Event();canceled=threading.Event();release=threading.Event();waiting=threading.Event()
        leases={'input':1,'key':1,'cleanup':0};snapshot_input=R.snapshot(bytearray([255]))
        state={};errors=[]
        def worker(token):
            started.set();assert canceled.wait(5);assert release.wait(5)
            assert leases['input']==leases['key']==1 and snapshot_input==b'\xff'
            leases['input']=leases['key']=0;leases['cleanup']+=1
            token.publish([]);token.acknowledge_cleanup()
        def run(main):
            s=R.sched();state['owner']=s;s.retiring_wait=waiting.set
            token=R.register_host(R.Task(10,Once(lambda t:None)),cancel=canceled.set)
            state['token']=token;threading.Thread(target=worker,args=(token,)).start();assert started.wait(5)
            if mode=='exception':raise ValueError('source step implementation fault')
            if mode=='fault':
                bad=R.register_host(main);bad.publish(fault='applicable adapter fault');bad.acknowledge_cleanup();s.drain()
        def drive():
            try:
                if mode=='reset':
                    R.reserve()
                    try:
                        s=R.Scheduler(R.Task(0,None),real=True);R._sched[0]=s
                        run(s.main)
                    finally:R.unreserve()
                    R.reset_scheduler()
                else:owner(run)
            except BaseException as e:errors.append(e)
        thread=threading.Thread(target=drive);thread.start()
        assert waiting.wait(5) and canceled.is_set()
        assert leases=={'input':1,'key':1,'cleanup':0} and thread.is_alive()
        release.set();thread.join(5);assert not thread.is_alive()
        assert leases=={'input':0,'key':0,'cleanup':1}
        assert not state['token'].publish([]) and not state['token'].acknowledge_cleanup()
        assert bool(errors)==(mode in ('exception','fault')), errors
    # Cleanup faults release every other context/task/operation registration.
    counts=[]
    def bad(main):
        s=R.sched()
        for i in range(3):
            token=R.register_host(R.Task(i+1,None),cleanup=lambda i=i:(counts.append(i),(_ for _ in ()).throw(ValueError('cleanup')))[1])
            token.acknowledge_cleanup()
        c,_=std_context_with_cancel(C.BACKGROUND)
        c.hooks.add(lambda:(_ for _ in ()).throw(ValueError('cancel hook')))
        c.hooks.add(lambda:counts.append(3))
        try:C.cancel_context(c,C.CONTEXT_CANCELED)
        except R.HostFault:pass
        assert 3 in counts
    check_fault(lambda:owner(bad),'cleanup');assert set(counts)=={0,1,2,3}


def real_progress_stress():
    results=[];clean=[];started=threading.Event();release=threading.Event()
    progress_started=threading.Event();progress=[0]
    executor=concurrent.futures.ThreadPoolExecutor(max_workers=8)
    class Child(R.Frame):
        def __init__(self,i,wg):super().__init__();self.i=i;self.wg=wg
        def step(self,t):
            if self.pc==0:
                self.pc=1;i=self.i
                def work(copied):
                    assert copied==[bytes([i%256])]
                    started.set();assert release.wait(5);return [i,bytearray([255,128])]
                R.launch_host(t,lambda copy:executor.submit(work,copy),inputs=[bytearray([i%256])],cleanup=lambda:clean.append(i))
            else:
                assert t.rv==[self.i,b'\xff\x80'];results.append(self.i)
                std_sync_waitgroup_add(self.wg,-1);R.ret(t,self)
    class Progress(R.Frame):
        def step(self,t):
            progress_started.set();progress[0]+=1
            if release.is_set():R.ret(t,self)
            else:R.yield_task(t)
    class Main(R.Frame):
        def step(self,t):
            if self.pc==0:
                R.spawn(Progress())
                self.pc=1;self.wg=WaitGroup();std_sync_waitgroup_add(self.wg,256)
                for i in range(256):R.spawn(Child(i,self.wg))
                R.yield_task(t)
            elif self.pc==1:
                assert started.wait(5) and progress_started.is_set() # barriers before measured deadline
                self.context_parent,self.cancel_parent=std_context_with_timeout(C.BACKGROUND,1000000000)
                self.child,_=std_context_with_timeout(self.context_parent,20000000)
                self.pc=2;self.at=time.monotonic_ns();std_time_sleep(t,21000000)
            elif self.pc==2:
                assert time.monotonic_ns()-self.at>=21000000 and progress[0]>1
                assert self.child.err is C.CONTEXT_DEADLINE_EXCEEDED and self.context_parent.err is None
                self.cancel_parent()
                release.set();self.pc=3;std_sync_waitgroup_wait(t,self.wg)
            else:
                assert sorted(results)==list(range(256)) and sorted(clean)==list(range(256))
                assert len(R.sched().tasks)==1 and not R.sched().operations
                R.ret(t,self)
    try:R.run_main_host(Main)
    finally:release.set();executor.shutdown(wait=True)


def anchored_boundary_future_lifetime():
    started=threading.Event();release=threading.Event();cancelled=threading.Event();waiting=threading.Event()
    f=concurrent.futures.Future();state={'input':R.snapshot(bytearray([255])),'key':object()};cancel_result=[]
    def worker():
        assert f.set_running_or_notify_cancel()
        started.set();assert cancelled.wait(5);assert release.wait(5)
        assert state['input']==b'\xff' and state['key'] is not None
        state.clear();f.set_result([]) # callbacks may run here, after resource release
    threading.Thread(target=worker).start();assert started.wait(5)
    def run(main):
        R.sched().retiring_wait=waiting.set
        def cancel():
            cancel_result.append(f.cancel());cancelled.set()
        R.launch_host(R.Task(10,None),lambda _:f,cancel=cancel)
    errors=[]
    def drive():
        try:owner(run)
        except BaseException as e:errors.append(e)
    driver=threading.Thread(target=drive);driver.start();assert waiting.wait(5)
    assert cancel_result==[False] and not f.done() and not f.cancelled() and state and driver.is_alive()
    release.set();driver.join(5);assert not driver.is_alive() and not errors and not state
    # Captured request deadline expires during source work before submission.
    barrier=threading.Event();threading.Thread(target=barrier.set).start();assert barrier.wait(5)
    submitted=[]
    class Boundary(R.Frame):
        def step(self,t):
            if self.pc==0:
                self.request=C.host_boundary(C.BACKGROUND,10000000)
                self.at=self.request.deadline;self.pc=1;std_time_sleep(t,12000000)
            elif self.pc==1:
                assert self.request.err is C.CONTEXT_DEADLINE_EXCEEDED and self.request.deadline==self.at
                self.pc=2
                def submit(copy):
                    submitted.append(1);f=concurrent.futures.Future();f.set_result([b'late success']);return f
                R.launch_host(t,submit,self.request,cancel_result=lambda err:[err])
            else:
                assert t.rv==[C.CONTEXT_DEADLINE_EXCEEDED] and C.BACKGROUND.err is None
                R.ret(t,self)
    R.run_main_host(Boundary);assert submitted==[1]
    @source_guard
    def recursive(n):
        if n:return recursive(n-1)+1
        return 0
    @source_guard
    def failure():raise ValueError('ordinary exception')
    def balanced(t):
        assert recursive(20)==20 and source_guard_state.depth==0
        try:failure()
        except ValueError:pass
        assert source_guard_state.depth==0
    owner(balanced)


def fatal_mode(mode,marker):
    def run(main):
        s=R.sched()
        if mode!='deadlock':
            cancel=threading.Event();token=R.register_host(R.Task(1,None),cancel=cancel.set)
            def work():
                assert cancel.wait(5)
                assert sys.stdin.readline().strip()=='release'
                Path(marker).write_text(mode+':clean')
                token.acknowledge_cleanup()
            threading.Thread(target=work).start()
            s.retiring_wait=lambda:print('owner waiting for cleanup ACK',flush=True)
        if mode=='panic':raise go_panic(box(STRING_TYPE,b'source failure'))
        if mode=='mutex-fatal':R.fatal('sync: unlock of unlocked mutex')
        if mode=='deadlock':
            Path(marker).write_text(mode+':clean');s.block(main)
    if mode=='deadlock':
        class Dead(R.Frame):
            def step(self,t):run(t)
        R.run_main_host(Dead)
    else:owner(run)


if __name__=='__main__':
    if len(sys.argv)>1:fatal_mode(sys.argv[1],sys.argv[2])
    else:
        before=(sys.getrecursionlimit(),threading.stack_size())
        for fn in [identity_fifo_wire,cancellation_faults,context_timers,foreign_cancel_thread_checks,blocked_roots,cleanup_leases,real_progress_stress,anchored_boundary_future_lifetime]:
            fn();print('PASS '+fn.__name__)
        assert before==(sys.getrecursionlimit(),threading.stack_size())
        assert source_guard_state.depth==0
        print('PASS actual Python host lifecycle')
