//! Tests compile the actual Rust runtime. Native work never receives source V.
mod rt;
use rt::*;
use std::sync::{Arc, Barrier, atomic::{AtomicUsize, AtomicBool, Ordering}};
use std::time::{Duration, Instant};
fn canceled(e: V) -> Vec<V> { vec![V::Nil, e] }
fn decode(w: Vec<HostWire>) -> Vec<V> { w.into_iter().map(|w| match w { HostWire::Int(i) => V::Int(i), HostWire::Bytes(b) => to_bytes(s(&b)), _ => V::Nil }).collect() }
fn entry_step(t: &Rc<Task>, f: &Rc<Frame>) {
    if f.pc.get() == 0 {
        f.pc.set(1); let init = f.prim.borrow_mut().take().unwrap(); init(t); return
    }
    let after = f.prim.borrow_mut().take(); if let Some(after) = after { after(t); }
    ret(t, f);
}
fn scenario(start: impl FnOnce(&Rc<Task>) + 'static, after: impl FnOnce(&Rc<Task>) + 'static) -> Result<(), HostError> {
    run_host_frame(|| {
        let frame = Frame::new(0, entry_step, None); let weak = Rc::downgrade(&frame);
        frame.prim.replace(Some(Box::new(move |t| {
            weak.upgrade().unwrap().prim.replace(Some(Box::new(after))); start(t);
        }))); frame
    })
}
fn register(t: &Rc<Task>, ctx: V, cleanups: Arc<AtomicUsize>) -> HostToken {
    register_host(t, host_boundary(ctx, None), vec![], canceled, decode, || {}, move || { cleanups.fetch_add(1, Ordering::SeqCst); })
}
#[test]
fn reentrant_wire_ids_ack_and_cancellation_order() {
    let count = Arc::new(AtomicUsize::new(0)); let start_count = count.clone();
    scenario(move |t| {
        let token = register(t, background(), start_count);
        assert!(t.blocked.get()); assert!(t.rv.borrow().is_empty());
        let mut wrong = token.clone(); wrong.task += 1;
        wrong.fault("foreign task fault"); wrong.acknowledge_cleanup();
        assert_eq!(token.mailbox.counts(), (1,0,0));
        let mut bytes = vec![0,255,128]; token.complete(vec![HostWire::Bytes(bytes.clone())]); bytes[0] = 99;
        token.complete(vec![HostWire::Int(99)]); wrong.acknowledge_cleanup();
        assert_eq!(token.mailbox.counts(), (1,1,0)); token.acknowledge_cleanup();
        assert!(t.rv.borrow().is_empty());
    }, |t| { assert_eq!(from_bytes(t.rv(0)).bytes().as_ref(), &[0,255,128]); }).unwrap();
    assert_eq!(count.load(Ordering::SeqCst),1);
    for fault in [false,true] {
        let ctx = Rc::new(RefCell::new(V::Nil)); let first = ctx.clone(); let second = ctx.clone();
        let result = scenario(move |t| {
            let c = new_child(background()); first.replace(c.clone());
            let token = register(t,c.clone(),Arc::new(AtomicUsize::new(0)));
            token.complete(if fault { vec![] } else { vec![HostWire::Int(7)] });
            // A fault can be the first applicable record, never overwrite success.
            if fault { /* tested separately below */ }
            cancel_ctx(c,context_canceled()); token.acknowledge_cleanup();
        }, move |t| { assert!(t.rv(0).is_nil()); assert!(veq(&t.rv(1), &context_canceled())); assert!(!second.borrow().is_nil()); });
        assert_eq!(result,Ok(()));
    }
    let fault = scenario(|t| {
        let c = new_child(background()); let token = register(t,c.clone(),Arc::new(AtomicUsize::new(0)));
        cancel_ctx(c,context_canceled()); token.fault("applicable adapter fault"); token.acknowledge_cleanup();
    }, |_| panic!("fault resumed source"));
    assert_eq!(fault,Err(HostError::Fault("applicable adapter fault".into())));
    // Committed success stays final after later source cancellation.
    let c = Rc::new(RefCell::new(V::Nil)); let first=c.clone();
    scenario(move |t| { let ctx=new_child(background()); first.replace(ctx.clone()); let token=register(t,ctx,Arc::new(AtomicUsize::new(0))); token.complete(vec![HostWire::Int(9)]); token.acknowledge_cleanup(); }, move |t| { cancel_ctx(c.borrow().clone(),context_canceled()); assert_eq!(t.rv(0).i(),9); }).unwrap();
}
#[test]
fn real_mailbox_wake_and_request_timeout_anchored_before_submission() {
    let started = Arc::new(Barrier::new(2)); let gate=started.clone(); let count=Arc::new(AtomicUsize::new(0)); let c=count.clone();
    let before=Instant::now();
    scenario(move |t| {
        let token=register(t,background(),c); launch_host(token,move || { gate.wait(); vec![HostWire::Int(73)] });
        started.wait(); // proves worker actually started before source parks
    }, |t| assert_eq!(t.rv(0).i(),73)).unwrap();
    assert!(before.elapsed()<Duration::from_secs(5)); assert_eq!(count.load(Ordering::SeqCst),1);
    scenario(|t| {
        let parent=new_child(background()); let boundary=host_boundary(parent.clone(),Some(1));
        while clock_now() < boundary.deadline.unwrap() { std::hint::spin_loop(); }
        let token=register_host(t,boundary,vec![parent],canceled,decode,|| {},|| {});
        launch_host(token,|| vec![HostWire::Int(99)]);
    }, |t| { assert!(t.rv(0).is_nil()); assert!(veq(&t.rv(1),&context_deadline_exceeded())); }).unwrap();
}
#[test]
fn absolute_contexts_hooks_and_repeated_pruning() {
    assert_eq!(elapsed_nanos(u128::MAX,8),i64::MAX); assert_eq!(elapsed_nanos(3,8),8);
    assert_eq!(deadline_after(i64::MAX-4,5),i64::MAX);
    let callbacks=Arc::new(AtomicUsize::new(0));
    for _ in 0..8 {
        let callbacks=callbacks.clone();
        scenario(move |t| {
            let bg=background(); let h=add_context_hook(bg.clone(),vec![byte_array(vec![8;65536])],|| panic!("Background hook retained")); h.remove();
            assert_eq!(raw_ctx(&bg,|c| c.hooks.len()),0);
            let now=clock_now(); let parent=new_child_deadline(bg.clone(),Some(now.saturating_add(1_000_000_000)));
            let child=new_child_deadline(parent.clone(),Some(i64::MAX));
            assert_eq!(raw_ctx(&parent,|c| c.deadline),raw_ctx(&child,|c| c.deadline));
            assert_eq!(sched(|s| s.timers.iter().find(|tm| tm.f.as_ref().map_or(false,|(_,v)| veq(v,&child))).unwrap().at),raw_ctx(&child,|c| c.deadline.unwrap()));
            let expired=new_child_deadline(bg.clone(),Some(clock_now()));
            assert!(veq(&std_context_context_err(expired.clone()),&context_deadline_exceeded()));
            let nested=new_child(expired); assert!(veq(&std_context_context_err(nested),&context_deadline_exceeded()));
            for context in [parent.clone(),child.clone()] { let cb=callbacks.clone(); add_context_hook(context,vec![],move || { cb.fetch_add(1,Ordering::SeqCst); }); }
            cancel_ctx(parent,context_canceled());
            assert_eq!(sched(|s| s.contexts.len()),0); assert_eq!(sched(|s| s.timers.len()),0);
            let token=register(t,bg,Arc::new(AtomicUsize::new(0))); token.complete(vec![]); token.acknowledge_cleanup();
        }, |_| {}).unwrap();
        assert_eq!(sched(|s| (s.contexts.len(),s.timers.len(),s.tasks.len(),s.pending.len(),s.host_roots.len())),(0,0,0,0,0));
    }
    assert_eq!(callbacks.load(Ordering::SeqCst),16);
    let stale=Rc::new(RefCell::new(V::Nil)); let first=stale.clone();
    scenario(move |t| { first.replace(new_child(background())); let token=register(t,background(),Arc::new(AtomicUsize::new(0))); token.complete(vec![]); token.acknowledge_cleanup(); }, |_| {}).unwrap();
    assert!(native_catch(|| { std_context_context_err(stale.borrow().clone()); }).unwrap_err().contains("retired"));
    assert!(catch(|| std_context_context_err(V::Nil)).is_err());
}
#[test]
fn constructor_failure_releases_reservation() {
    assert_eq!(run_host_frame(|| host_fault("constructor")),Err(HostError::Fault("constructor".into())));
    scenario(|_| {}, |_| {}).unwrap();
    let r=reserve_entry().unwrap();
    assert_eq!(run_host_frame(|| panic!("overlap touched constructor")),Err(HostError::Fault("overlapping executable entry/reset".into())));
    assert!(native_catch(reset_scheduler).is_err()); drop(r);
    scenario(|_| {}, |_| {}).unwrap();
}
#[test]
fn owned_globals_keep_only_latest_without_truncating_other_roots() {
    for n in [1,1,2,1,3,1] {
        init_globals(n);
        gs(0,byte_array(vec![n as u8;65536]));
        let frame=Fr::new(1); frame.s(0,byte_array(vec![255;10]));
        init_globals(n); collect();
        assert_eq!(slot(frame.g(0).h(),0).i(),255);
        assert_eq!(slot(gg(0).h(),0).i(),n as i64);
    }
    collect(); assert_eq!(heap_live(),1);
}
#[test]
fn pending_roots_decoder_outputs_and_forced_safepoints() {
    scenario(|t| {
        let input=byte_array(vec![255;65536]); let key=cellv(V::Int(81));
        let capture=input.clone(); let key_capture=key.clone();
        let token=register_host(t,host_boundary(background(),None),vec![input,key],canceled,move |_| {
            collect(); assert_eq!(slot(capture.h(),65535).i(),255); assert_eq!(pget(&key_capture).i(),81);
            let output=to_bytes(s(b"\0\xff\x80"));
            // Decoder partial outputs use ordinary synchronous roots at safepoints.
            let _root=temp_root(&[output.clone()]); collect(); vec![output]
        },|| {},|| collect());
        collect(); token.complete(vec![]); token.acknowledge_cleanup();
    }, |t| { collect(); assert_eq!(from_bytes(t.rv(0)).bytes().as_ref(),&[0,255,128]); }).unwrap();
    collect(); let baseline=heap_live();
    for _ in 0..32 {
        scenario(|t| { let input=byte_array(vec![8;10000]); let token=register_host(t,host_boundary(background(),None),vec![input],canceled,decode,|| {},|| collect()); token.complete(vec![]); token.acknowledge_cleanup(); }, |_| {}).unwrap();
        collect(); assert_eq!(heap_live(),baseline);
    }
}
#[test]
fn each_blocked_root_detaches_with_external_object_retained() {
    for kind in 0..7 {
        let object=Rc::new(RefCell::new(V::Nil)); let saved=Rc::new(RefCell::new(None::<Waiter>)); let weak=Rc::new(RefCell::new(None::<std::rc::Weak<Task>>));
        let o=object.clone(); let w=saved.clone(); let weak_save=weak.clone();
        scenario(move |main| {
            let frame=Frame::new(0,entry_step,None);
            frame.prim.replace(Some(Box::new(move |t| {
                weak_save.replace(Some(Rc::downgrade(t)));
                let obj=match kind { 4=>new_mutex(), 5=>new_waitgroup(), _=>make_chan(V::Int(0),|| V::Nil) };
                o.replace(obj.clone());
                match kind {
                    0=>chan_send(t,obj.clone(),byte_array(vec![255;65536])),
                    1=>chan_recv(t,obj.clone()),
                    2=>select(t,false,vec![scase(obj.clone(),true,byte_array(vec![128;65536]))]),
                    3=>select(t,false,vec![scase(obj.clone(),false,V::Nil)]),
                    4=>{ with_mutex(&obj,|m| m.locked=true); std_sync_mutex_lock(t,obj.clone()); assert_eq!(with_mutex(&obj,|m| m.waiters.len()),1); },
                    5=>{std_sync_waitgroup_add(obj.clone(),V::Int(1)); std_sync_waitgroup_wait(t,obj.clone()); assert_eq!(with_wg(&obj,|wg| wg.waiters.len()),1);},
                    _=>std_time_sleep(t,V::Int(i64::MAX)),
                }
                if kind<4 { w.replace(with_chan(&obj,|c| { assert_eq!(c.sendq.len()+c.recvq.len(),1); c.sendq.front().or(c.recvq.front()).cloned() })); }
            })));
            spawn(V::Frame(frame)); yield_task(main);
        }, |_| {}).unwrap();
        let retained=object.borrow().clone(); let _root=temp_root(&[retained.clone()]); collect();
        if kind<4 {
            assert_eq!(with_chan(&retained,|c| (c.sendq.len(),c.recvq.len())),(0,0));
            let stale=saved.borrow().clone().unwrap(); assert!(stale.val.borrow().is_nil()); assert!(!stale.live()); recv_done(stale.clone(),V::Int(99),true); send_done(stale,true);
        } else if kind==4 { assert_eq!(with_mutex(&retained,|m| m.waiters.len()),0); }
        else if kind==5 { assert_eq!(with_wg(&retained,|w| w.waiters.len()),0); }
        assert!(weak.borrow().as_ref().unwrap().upgrade().is_none(),"task Rc retained for kind {}",kind);
    }
}
#[test]
fn retirement_waits_real_resource_ack_and_exhausts_faults() {
    let lease=Arc::new(AtomicBool::new(true)); let released=lease.clone(); let cancel=Arc::new(Barrier::new(2)); let signal=cancel.clone();
    let cleanup=Arc::new(AtomicUsize::new(0)); let clean=cleanup.clone();
    let result=scenario(move |t| {
        let token=register_host(t,host_boundary(background(),None),vec![byte_array(vec![9;65536])],canceled,decode,move || { signal.wait(); host_fault("cancel hook fault") },move || {clean.fetch_add(1,Ordering::SeqCst);collect();host_fault("cleanup hook fault")});
        let inspect=token.clone();
        std::thread::spawn(move || {
            cancel.wait(); while !inspect.mailbox.waiting_cleanup.load(Ordering::Acquire) { std::thread::yield_now(); }
            assert!(released.load(Ordering::SeqCst)); released.store(false,Ordering::SeqCst); inspect.acknowledge_cleanup();
            inspect.complete(vec![HostWire::Int(1)]); // shutdown does not resume source
        });
        t.blocked.set(false); // main return while pending lease is still acquired
    }, |_| {});
    assert_eq!(result,Err(HostError::Fault("cancel hook fault".into()))); assert!(!lease.load(Ordering::SeqCst)); assert_eq!(cleanup.load(Ordering::SeqCst),1);
    assert_eq!(sched(|s| s.mailbox.counts()),(0,0,0));
    scenario(|_| {}, |_| {}).unwrap();
}
static STRESS_RESULTS: AtomicUsize=AtomicUsize::new(0);
static STRESS_CLEANUPS: AtomicUsize=AtomicUsize::new(0);
fn stress_op(t:&Rc<Task>,f:&Rc<Frame>) {
    if f.pc.get()==0 { f.pc.set(1); let i=f.l.g(0).i(); let token=register_host(t,host_boundary(background(),None),vec![],canceled,decode,|| {},|| {STRESS_CLEANUPS.fetch_add(1,Ordering::SeqCst);}); launch_host(token,move || vec![HostWire::Int(i)]); return }
    assert_eq!(t.rv(0).i(),f.l.g(0).i()); STRESS_RESULTS.fetch_add(1,Ordering::SeqCst); std_sync_waitgroup_done(f.l.g(1)); ret(t,f);
}
fn stress_main(t:&Rc<Task>,f:&Rc<Frame>) {
    if f.pc.get()==0 {
        f.pc.set(1); let wg=new_waitgroup(); f.l.s(0,wg.clone()); std_sync_waitgroup_add(wg.clone(),V::Int(256));
        for i in 0..256 { let f=Frame::new(2,stress_op,None);f.l.s(0,V::Int(i));f.l.s(1,wg.clone());spawn(V::Frame(f)); }
        std_sync_waitgroup_wait(t,wg);return
    }
    assert_eq!(STRESS_RESULTS.load(Ordering::SeqCst),256); assert_eq!(STRESS_CLEANUPS.load(Ordering::SeqCst),256); assert_eq!(sched(|s| s.pending.len()),0); ret(t,f);
}
#[test]
fn bounded_stress_asserts_every_source_resume_before_main_return() {
    STRESS_RESULTS.store(0,Ordering::SeqCst);STRESS_CLEANUPS.store(0,Ordering::SeqCst);
    run_host_frame(|| Frame::new(1,stress_main,None)).unwrap();
}
#[test]
fn worker_fault_separate_from_source_panic_and_declared_error() {
    let r=scenario(|t| { let token=register(t,background(),Arc::new(AtomicUsize::new(0))); launch_host(token,|| panic!("unexpected worker")); }, |_| panic!("fault resumed"));
    assert_eq!(r,Err(HostError::Fault("unexpected worker".into())));
    let r=scenario(|_| throw(boxv(&STRING_TYPE,s(b"source panic"))), |_| {});
    assert_eq!(r,Err(HostError::SourcePanic(b"panic: source panic\n".to_vec())));
    scenario(|t| { let token=register(t,background(),Arc::new(AtomicUsize::new(0))); token.complete(vec![HostWire::Bytes(b"declared transport error".to_vec())]); token.acknowledge_cleanup(); }, |t| assert_eq!(from_bytes(t.rv(0)).bytes().as_ref(),b"declared transport error")).unwrap();
}
thread_local! { static TIMERS: RefCell<Vec<i64>>=RefCell::new(Vec::new()); }
fn timer_record(v:V) { TIMERS.with(|t| t.borrow_mut().push(v.i())); }
#[test]
fn controlled_absolute_timers_equal_overflow_and_order() {
    scenario(|t| {
        let epoch=sched(|s| {let e=s.epoch.take();s.clock=100;e});
        add_timer_at(101,None,Some((timer_record,V::Int(2))));
        add_timer_at(100,None,Some((timer_record,V::Int(0))));
        add_timer_at(101,None,Some((timer_record,V::Int(3))));
        add_timer_at(100,None,Some((timer_record,V::Int(1))));
        let first=new_child_deadline(background(),Some(777)); let second=new_child_deadline(first.clone(),Some(999));
        assert_eq!(raw_ctx(&second,|c| c.deadline),Some(777));
        assert_eq!(sched(|s| s.timers.iter().find(|tm| tm.f.as_ref().map_or(false,|(_,v)|veq(v,&second))).unwrap().at),777);
        cancel_ctx(first,context_canceled());
        sched(|s| s.clock=101); fire_due_host_timers();
        TIMERS.with(|v|assert_eq!(*v.borrow(),vec![0,1,2,3]));
        sched(|s| s.clock=i64::MAX-1); add_timer(3,None,Some((timer_record,V::Int(4))));
        assert_eq!(sched(|s| s.timers[0].at),i64::MAX);
        sched(|s| {s.timers.clear();s.clock=0;s.epoch=epoch;});
        let token=register(t,background(),Arc::new(AtomicUsize::new(0)));token.complete(vec![]);token.acknowledge_cleanup();
    }, |_| {}).unwrap();
}
#[test]
fn publication_fifo_survives_ack_ready_hole_and_out_of_order_registration() {
    let order=Rc::new(RefCell::new(Vec::new()));let result=order.clone();
    scenario(move |main| {
        let wg=new_waitgroup();std_sync_waitgroup_add(wg.clone(),V::Int(3));
        let tokens=Rc::new(RefCell::new(Vec::<HostToken>::new()));
        for i in 1..=3 {
            let frame=Frame::new(0,entry_step,None);frame.pc.set(1);
            let task=sched(|s| {let t=Task::new(s.next_id,Some(frame.clone()));s.next_id+=1;t.owner.set(s.owner);s.tasks.push(t.clone());t});
            let seen=order.clone();let group=wg.clone();let token_ref=tokens.clone();
            frame.prim.replace(Some(Box::new(move |t| {
                assert_eq!(t.rv(0).i(),i); seen.borrow_mut().push(i);
                if i==2 {let ts=token_ref.borrow();ts[2].complete(vec![HostWire::Int(3)]);ts[0].acknowledge_cleanup();}
                std_sync_waitgroup_done(group);
            })));
            task.cleanup_roots.replace(vec![wg.clone()]); // owner traces frame callback group capture
            let tok=register_host(&task,host_boundary(background(),None),vec![wg.clone()],canceled,decode,|| {},|| {});tokens.borrow_mut().push(tok);
        }
        {let ts=tokens.borrow();ts[0].complete(vec![HostWire::Int(1)]);ts[1].complete(vec![HostWire::Int(2)]);ts[1].acknowledge_cleanup();ts[2].acknowledge_cleanup();}
        std_sync_waitgroup_wait(main,wg);
    }, |_| {}).unwrap();
    assert_eq!(*result.borrow(),vec![2,1,3]);
}
#[test]
fn stale_tokens_and_hooks_cannot_revive_or_retain_new_owner() {
    let saved=Rc::new(RefCell::new(None::<HostToken>));let first=saved.clone();
    let remover=Rc::new(RefCell::new(None::<ContextHookHandle>));let handle=remover.clone();
    scenario(move |t| {
        handle.replace(Some(add_context_hook(background(),vec![],|| {})));
        let token=register(t,background(),Arc::new(AtomicUsize::new(0)));first.replace(Some(token.clone()));token.complete(vec![]);token.acknowledge_cleanup();
    }, |_| {}).unwrap();
    for _ in 0..1000 {let token=saved.borrow();let token=token.as_ref().unwrap();token.fault("stale");token.acknowledge_cleanup();}
    assert_eq!(saved.borrow().as_ref().unwrap().mailbox.counts(),(0,0,0));
    scenario(move |t| {
        assert!(native_catch(||remover.borrow_mut().take().unwrap().remove()).is_err());
        let old=saved.borrow().as_ref().unwrap().clone();let token=register(t,background(),Arc::new(AtomicUsize::new(0)));old.complete(vec![HostWire::Int(99)]);
        assert_eq!(token.mailbox.counts(),(1,0,0));token.complete(vec![HostWire::Int(4)]);token.acknowledge_cleanup();
    }, |t|assert_eq!(t.rv(0).i(),4)).unwrap();
}
#[test]
fn all_context_hooks_remain_traced_and_cleanup_exhausts_decode_fault() {
    let hooks=Arc::new(AtomicUsize::new(0));let count=hooks.clone();
    let r=scenario(move |t| {
        let c=new_child(background());
        let value=byte_array(vec![171;1024]);let v=value.clone();let h=count.clone();
        add_context_hook(c.clone(),vec![],|| {collect();host_fault("first hook");});
        add_context_hook(c,vec![value],move || {assert_eq!(slot(v.h(),0).i(),171);h.fetch_add(1,Ordering::SeqCst);});
        let token=register_host(t,host_boundary(background(),None),vec![],canceled,|_| host_fault("decoder fault"),|| {},|| collect());
        token.complete(vec![]);token.acknowledge_cleanup();
    }, |_|panic!("decoder fault resumed"));
    assert_eq!(r,Err(HostError::Fault("decoder fault".into())));assert_eq!(hooks.load(Ordering::SeqCst),1);
}
#[test]
fn nested_cancellation_roots_detached_parent_hooks_before_child_collect() {
    scenario(|_| {
        let parent=new_child(background());let child=new_child(parent.clone());
        let payload=to_bytes(s(b"parent capture"));let captured=payload.clone();
        add_context_hook(parent.clone(),vec![payload],move || assert_eq!(from_bytes(captured).bytes().as_ref(),b"parent capture"));
        add_context_hook(child,vec![],|| collect());
        cancel_ctx(parent,context_canceled());
    }, |_| {}).unwrap();
}
fn collected_timer(_:V){collect()}
fn captured_timer(v:V){assert_eq!(from_bytes(v).bytes().as_ref(),b"detached timer payload")}
#[test]
fn all_detached_due_timers_remain_traced_across_first_callback_collection() {
    scenario(|_| {
        add_timer_at(0,None,Some((collected_timer,V::Nil)));
        add_timer_at(0,None,Some((captured_timer,to_bytes(s(b"detached timer payload")))));
        fire_due_host_timers();
    }, |_| {}).unwrap();
}
#[test]
fn retirement_detaches_defer_runner_child_parent_rc_cycles() {
    let frames=Rc::new(RefCell::new(Vec::<std::rc::Weak<Frame>>::new()));let saved=frames.clone();
    let result=scenario(move |t| {
        let runner=t.frame.borrow().clone().unwrap();let child=Frame::new(1,entry_step,None);
        runner.b.replace(Some(child.clone()));child.parent.replace(Some(runner.clone()));child.l.s(0,byte_array(vec![7;65536]));
        saved.borrow_mut().extend([Rc::downgrade(&runner),Rc::downgrade(&child)]);
        host_fault("deferred child active");
    }, |_| {});
    assert_eq!(result,Err(HostError::Fault("deferred child active".into())));
    assert!(frames.borrow().iter().all(|f|f.upgrade().is_none()));collect();
}
#[test]
fn poison_and_native_raii_fault_cleanup_do_not_strand_ack() {
    let released=Arc::new(AtomicBool::new(false));let flag=released.clone();
    struct Lease(Arc<AtomicBool>);impl Drop for Lease{fn drop(&mut self){self.0.store(true,Ordering::SeqCst);}}
    let result=scenario(move|t|{
        let token=register(t,background(),Arc::new(AtomicUsize::new(0)));let mailbox=token.mailbox.clone();
        launch_host(token,move||{let _lease=Lease(flag);let _lock=mailbox.state.lock().unwrap();panic!("poison after native acquisition")});
    }, |_|panic!("poison resumed source"));
    assert!(matches!(result,Err(HostError::Fault(_))));assert!(released.load(Ordering::SeqCst));
    assert_eq!(sched(|s|s.mailbox.counts()),(0,0,0));scenario(|_|{}, |_|{}).unwrap();
}
static TICKS:AtomicUsize=AtomicUsize::new(0);
static NATIVE_STARTED:AtomicBool=AtomicBool::new(false);
fn runnable_progress(t:&Rc<Task>,_:&Rc<Frame>){TICKS.fetch_add(1,Ordering::SeqCst);yield_task(t)}
fn real_deadline_main(t:&Rc<Task>,f:&Rc<Frame>){
    match f.pc.get(){
        0=>{
            f.pc.set(1);TICKS.store(0,Ordering::SeqCst);NATIVE_STARTED.store(false,Ordering::SeqCst);
            spawn(V::Frame(Frame::new(0,runnable_progress,None)));
            let native=Frame::new(0,entry_step,None);
            native.prim.replace(Some(Box::new(|t|{
                let cancel=Arc::new(AtomicBool::new(false));let signal=cancel.clone();
                let tok=register_host(t,host_boundary(background(),None),vec![],canceled,decode,move||{signal.store(true,Ordering::SeqCst);},||{});
                launch_host(tok,move||{NATIVE_STARTED.store(true,Ordering::Release);while !cancel.load(Ordering::SeqCst){std::thread::yield_now()}vec![]});
            })));spawn(V::Frame(native));yield_task(t);
        }
        1=>{
            if !NATIVE_STARTED.load(Ordering::Acquire)||TICKS.load(Ordering::SeqCst)==0{yield_task(t);return}
            // Both native acquisition and source progress precede the measured
            // deadline; scheduler load cannot invalidate this barrier.
            f.pc.set(2);let begin=clock_now();f.l.s(0,V::Int(begin));f.l.s(1,V::Int(TICKS.load(Ordering::SeqCst)as i64));
            let parent=new_child_deadline(background(),Some(begin+5_000_000));let child=new_child_deadline(parent.clone(),Some(begin+10_000_000));f.l.s(2,parent);f.l.s(3,child.clone());
            chan_recv(t,std_context_context_done(child));
        }
        _=>{
            assert!(clock_now()-f.l.g(0).i()>=5_000_000);assert!(TICKS.load(Ordering::SeqCst)>=f.l.g(1).i()as usize); // start barrier, not load-sensitive extra ticks
            assert!(veq(&std_context_context_err(f.l.g(2)),&context_deadline_exceeded()));assert!(veq(&std_context_context_err(f.l.g(3)),&context_deadline_exceeded()));ret(t,f);
        }
    }
}
#[test]
fn due_real_timers_dispatch_while_source_runnable_and_native_pending(){run_host_frame(||Frame::new(4,real_deadline_main,None)).unwrap()}
#[test]
fn copied_input_and_owned_output_survive_alias_mutation_and_later_entry() {
    let saved=Rc::new(RefCell::new(Vec::new()));let output=saved.clone();
    scenario(|t| {
        let source=to_bytes(s(&[0,255,128,3]));let snapshot=from_bytes(source.clone()).bytes().to_vec();
        let token=register_host(t,host_boundary(background(),None),vec![source.clone()],canceled,decode,||{},||collect());
        sset(source,V::Int(0),V::Int(99));
        launch_host(token,move||{assert_eq!(snapshot,&[0,255,128,3]);vec![HostWire::Bytes(snapshot)]});
    },move|t|{let value=t.rv(0);output.replace(from_bytes(value.clone()).bytes().to_vec());sset(value,V::Int(1),V::Int(7));}).unwrap();
    scenario(|_|{byte_array(vec![8;10000]);collect();}, |_|{}).unwrap();assert_eq!(*saved.borrow(),vec![0,255,128,3]);
}
fn initialization_fault(){host_fault("owner initialization fault")}
fn simple_entry()->V{V::Frame(Frame::new(0,|t,f|ret(t,f),None))}
#[test]
fn dedicated_owner_initialization_fault_drops_reservation_before_reentry(){
    assert_eq!(run_main_host(0,initialization_fault,simple_entry),Err(HostError::Fault("owner initialization fault".into())));
    assert_eq!(run_main_host(0,||{},simple_entry),Ok(()));
    fn panic_entry()->V{throw(boxv(&STRING_TYPE,s(b"owned error after TLS")))}
    let result=run_main_host(0,||{},panic_entry);
    assert_eq!(result,Err(HostError::SourcePanic(b"panic: owned error after TLS\n".to_vec())));
    assert_eq!(run_main_host(0,||{},simple_entry),Ok(()));
}
#[test]
fn cancellation_result_decoder_fault_still_runs_acked_operation_cleanup(){
    fn failed_cancel(_:V)->Vec<V>{host_fault("cancellation result decode fault")}
    let cleanup=Arc::new(AtomicUsize::new(0));let count=cleanup.clone();
    let result=scenario(move|t|{
        let c=new_child(background());let token=register_host(t,host_boundary(c.clone(),None),vec![],failed_cancel,decode,||{},move||{collect();count.fetch_add(1,Ordering::SeqCst);});
        token.complete(vec![HostWire::Int(99)]);token.acknowledge_cleanup();cancel_ctx(c,context_canceled());
    }, |_|panic!("fault resumed source"));
    assert_eq!(result,Err(HostError::Fault("cancellation result decode fault".into())));assert_eq!(cleanup.load(Ordering::SeqCst),1);
    assert_eq!(sched(|s|(s.pending.len(),s.host_roots.len(),s.mailbox.counts())),(0,0,(0,0,0)));
}
fn pending_all_child(_: &[V], _:Vec<V>)->V{
    let f=Frame::new(0,entry_step,None);f.prim.replace(Some(Box::new(|t|chan_recv(t,V::Nil))));V::Frame(f)
}
#[test]
fn all_task_private_waitgroup_detaches_retained_group_and_weak_parent(){
    let group=Rc::new(RefCell::new(V::Nil));let retained=group.clone();
    let parent=Rc::new(RefCell::new(None::<std::rc::Weak<Task>>));let saved=parent.clone();
    scenario(|main|{
        let fns=V::Slice(alloc(Obj::Vals(vec![func(-1,pending_all_child,vec![])])),0,1,1);
        let caller=Frame::new(1,entry_step,None);caller.l.s(0,fns.clone());caller.prim.replace(Some(Box::new(move|t|lib_task_all(t,fns))));spawn(V::Frame(caller));yield_task(main);
    },move|_|{
        let (task,wg)=sched(|s|{
            let parent=s.tasks.iter().find(|t|t.id==1).unwrap().clone();let child=s.tasks.iter().find(|t|t.id==2).unwrap();let wg=child.frame.borrow().as_ref().unwrap().l.g(1);(parent,wg)
        });
        assert_eq!(with_wg(&wg,|g|g.waiters.len()),1);assert!(task.blocked.get());saved.replace(Some(Rc::downgrade(&task)));retained.replace(wg);
    }).unwrap();
    let wg=group.borrow().clone();let _root=temp_root(&[wg.clone()]);collect();assert_eq!(with_wg(&wg,|g|g.waiters.len()),0);assert!(parent.borrow().as_ref().unwrap().upgrade().is_none());
}
#[test]
fn stale_all_task_call_rejects_before_results_or_children_mutate(){
    let task=Rc::new(RefCell::new(None::<Rc<Task>>));let save=task.clone();
    scenario(move|t|{save.replace(Some(t.clone()));}, |_|{}).unwrap();
    let stale=task.borrow().as_ref().unwrap().clone();stale.set_rv(vec![V::Int(77)]);
    scenario(move|_|{
        let before=sched(|s|s.tasks.len());assert!(native_catch(||lib_task_all(&stale,V::Slice(0,0,0,0))).is_err());assert_eq!(stale.rv(0).i(),77);assert_eq!(sched(|s|s.tasks.len()),before);
    }, |_|{}).unwrap();
}
