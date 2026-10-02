/* Actual C runtime probes. Native jobs contain wire bytes/IDs only. */
#include "gx.h"
#include <assert.h>
#include <stdatomic.h>
#include <stdio.h>
#include <stdlib.h>

static int mode, decoded, cleaned, canceled, step_progress;
static atomic_int overlap_status;
static bool fail_frame;
bool test_only_fail_frame(void) {
    bool fail = fail_frame;
    fail_frame = false;
    return fail;
}
static void immediate_primitive(gx_Task *t, void *unused) {
    (void)unused;
    gx_V result = gx_int(42);
    gx_set_rv(t, 1, &result);
}
static void harness_constructor_failure(void) {
    gx_V out[16];
    fail_frame = true;
    gx_Handler h;
    if (!GX_TRY(h)) {
        gx_run_isolated(immediate_primitive, NULL, out);
        assert(!"constructor did not fail");
        GX_END(h);
    } else {
        assert(gx_thrown && gx_thrown->value.t == GX_STR);
        gx_thrown = NULL;
    }
    assert(gx_run_isolated(immediate_primitive, NULL, out) == 1 && out[0].u.i == 42);
}

static gx_V context, saved_context, saved_result, nested_children[2], retained_channel, retained_mutex,
    retained_waitgroup;
static gx_Waiter *retained_waiter;
static gx_HostToken tokens[300];
static const uint8_t binary[] = {0, 255, 128, 3};
static int registrations, order[300], expected[300];
static void **weak_due, **weak_child;
static int timer_callbacks;
static atomic_bool native_started;
static int busy_progress;
static void cancel_timer(gx_V c) { gx_cancel_ctx(c, gx_context_deadline_exceeded()); }

static void timer_order(gx_V arg) { order[timer_callbacks++] = (int)arg.u.i; }
static void timer_first(gx_V ignored) {
    (void)ignored;
    GC_gcollect();
    assert(*weak_due);
    assert(timer_callbacks++ == 0);
}
static void timer_second(gx_V root) {
    GC_gcollect();
    assert(root.l == 4 && !memcmp(gx_sbytes(root), binary, 4));
    assert(timer_callbacks++ == 1);
}

static atomic_int leases;
typedef struct NativeJob {
    gx_HostToken token;
    pthread_t thread;
    pthread_mutex_t mutex;
    pthread_cond_t gate;
    bool release, stop;
    uint8_t *input;
} NativeJob;
static NativeJob *native_job; /* native pointer only, no source graphs */
static const char *cleanup_count(void *ignored) {
    (void)ignored;
    cleaned++;
    return NULL;
}
static const char *cancel_count(void *ignored) {
    (void)ignored;
    canceled++;
    return NULL;
}
static const char *cancel_collect_fault(void *ignored) {
    canceled++;
    GC_gcollect();
    assert(*weak_child);
    gx_host_ack(tokens[(uintptr_t)ignored - 1]);
    if (canceled == 1)
        gx_host_fault("nested cancellation hook fault");
    return NULL;
}
static const char *cancel_fault(void *ignored) {
    (void)ignored;
    canceled++;
    gx_host_fault("cancel hook fault");
}
static const char *cleanup_fault(void *ignored) {
    (void)ignored;
    cleaned++;
    gx_host_fault("cleanup hook fault");
}
static const char *decode_bytes(gx_Task *t, const uint8_t *bytes, size_t n, gx_V err, gx_V roots) {
    GC_gcollect();
    assert(roots.t == GX_STR && roots.l == sizeof(binary));
    assert(!memcmp(gx_sbytes(roots), binary, sizeof(binary)));
    order[decoded++] = t->id;
    if (mode == 17)
        gx_throw(gx_cstr("cancellation decoder source panic"));
    if (mode == 5)
        gx_host_fault("decoder fault");
    if (mode == 3 || mode == 8 || mode == 9 || mode == 10 || mode == 15) {
        assert(err.t != GX_NIL);
        if (mode == 10) { assert(((gx_Context *)context.u.p)->err.t == GX_NIL); assert(err.u.p == gx_context_deadline_exceeded().u.p); }
        if (mode == 8 || mode == 9) {
            const gx_TypeDesc *type = gx_dyn_type(err); const gx_Method *method = gx_method(type, "Error"); assert(method);
            gx_V argument = gx_unboxed(err); gx_V text = method->code(NULL, &argument, 1);
            const char *category = mode == 8 ? "nil source context" : "foreign source context";
            assert(text.l == strlen(category) && !memcmp(gx_sbytes(text),category,text.l));
        }
        gx_V rv[] = {err};
        gx_set_rv(t, 1, rv);
        return NULL;
    }
    assert(err.t == GX_NIL && n == sizeof(binary) && !memcmp(bytes, binary, n));
    gx_V rv[] = {gx_str((const char *)bytes, n)};
    gx_set_rv(t, 1, rv);
    return NULL;
}
static gx_V entry(void);
static void init(void);
static void *native_work(void *arg) {
    NativeJob *j = arg;
    atomic_store(&overlap_status, gx_run_main_host(init, entry));
    atomic_fetch_add(&leases, 1);
    atomic_store(&native_started, true);
    pthread_mutex_lock(&j->mutex);
    while (!j->release && !j->stop)
        pthread_cond_wait(&j->gate, &j->mutex);
    pthread_mutex_unlock(&j->mutex);
    assert(!memcmp(j->input, binary, sizeof(binary)));
    gx_host_publish(j->token, j->input, sizeof(binary), NULL);
    free(j->input);
    j->input = NULL;
    atomic_fetch_sub(&leases, 1); /* actual native release precedes ACK */
    gx_host_ack(j->token);
    gx_host_token_release(j->token);
    return NULL;
}
static const char *native_cancel(void *arg) {
    NativeJob *j = arg;
    canceled++;
    pthread_mutex_lock(&j->mutex);
    j->stop = true;
    pthread_cond_broadcast(&j->gate);
    pthread_mutex_unlock(&j->mutex);
    return NULL;
}
static const char *native_cleanup(void *arg) {
    NativeJob *j = arg;
    GC_pthread_join(j->thread, NULL);
    assert(!j->input && atomic_load(&leases) == 0 && atomic_load(&overlap_status) == 1);
    pthread_cond_destroy(&j->gate);
    pthread_mutex_destroy(&j->mutex);
    free(j);
    cleaned++;
    native_job = NULL;
    return NULL;
}
static void release_native(void) {
    pthread_mutex_lock(&native_job->mutex);
    native_job->release = true;
    pthread_cond_broadcast(&native_job->gate);
    pthread_mutex_unlock(&native_job->mutex);
}
static void progress_step(gx_Task *t, gx_Frame *f) {
    step_progress++;
    release_native();
    gx_ret(t, f);
}
static void busy_step(gx_Task *t, gx_Frame *f) {
    if (f->pc == 0) {
        while (!atomic_load(&native_started)) {
        } /* acquired-work barrier before measured deadline */
        gx_add_timer_at(gx_deadline(gx_now(), 1000000), NULL, cancel_timer, context);
        f->pc = 1;
    }
    busy_progress++;
    if (((gx_Context *)context.u.p)->err.t == GX_NIL)
        gx_yield_task(t);
    else
        gx_ret(t, f);
}
static void child_step(gx_Task *t, gx_Frame *f) {
    if (f->pc++ == 0) {
        gx_HostToken token = gx_host_register(t, gx_host_boundary(mode == 30 ? f->l[0] : gx_background(), 0),
                                              gx_str((const char *)binary, sizeof(binary)), decode_bytes,
                                              mode == 30 ? cancel_collect_fault : cancel_count,
                                              mode == 16 && t->id == 2 ? cleanup_fault : cleanup_count,
                                              mode == 30 ? (void *)(uintptr_t)t->id : NULL);
        tokens[registrations++] = token;
        return;
    }
    assert(t->nrv == 1 && t->rv[0].l == sizeof(binary));
    gx_ret(t, f);
}
static void blocked_step(gx_Task *t, gx_Frame *f) {
    if (f->pc++ == 0) {
        switch (mode) {
        case 20:
            gx_chan_send(t, retained_channel, gx_str((const char *)binary, sizeof(binary)));
            retained_waiter = ((gx_Chan *)retained_channel.u.p)->sendq.head;
            break;
        case 21:
            gx_chan_recv(t, retained_channel);
            retained_waiter = ((gx_Chan *)retained_channel.u.p)->recvq.head;
            break;
        case 22: {
            gx_Case c = {retained_channel, true, gx_str((const char *)binary, sizeof(binary))};
            gx_select(t, false, 1, &c);
            retained_waiter = ((gx_Chan *)retained_channel.u.p)->sendq.head;
            break;
        }
        case 23:
            gx_std_sync_mutex_lock(t, retained_mutex);
            break;
        case 24:
            gx_std_sync_waitgroup_wait(t, retained_waitgroup);
            break;
        case 25:
            gx_chan_send(t, gx_nil(), gx_nil());
            break;
        case 26:
            gx_select(t, false, 0, NULL);
            break;
        case 27:
            gx_std_time_sleep(t, gx_int(INT64_MAX));
            break;
        }
        registrations++;
        return;
    }
    assert(!"retired blocked task executed");
}
static void *wrong_thread_waiter(void *unused) {
    (void)unused;
    gx_recv_done(retained_waiter, gx_int(7), true);
    gx_send_done(retained_waiter, true);
    if (retained_waiter->queue) {
        assert(gx_dequeue(retained_waiter->queue) == NULL);
        gx_enqueue(retained_waiter->queue, retained_waiter);
    }
    gx_V value; bool ok; assert(!gx_try_recv(retained_channel.u.p, &value, &ok));
    gx_detach_waiter(retained_waiter);
    gx_cancel_ctx(context, gx_context_canceled());
    return NULL;
}
static void ack_barrier_step(gx_Task *t, gx_Frame *f) {
    assert(decoded == 0 && cleaned == 0 && gx_host_live_count(tokens[0]) == 1);
    assert(gx_host_ack(tokens[0]));
    gx_ret(t, f);
}
static void root_step(gx_Task *t, gx_Frame *f) {
    if (mode == 30) {
        if (f->pc++ == 0) {
            context = gx_at(gx_std_context_with_cancel(gx_background()), 0);
            for (int i = 0; i < 2; i++) {
                nested_children[i] = gx_at(gx_std_context_with_cancel(context), 0);
                gx_Frame *child = gx_new_frame(1, child_step, NULL);
                child->l[0] = nested_children[i];
                gx_spawn(gx_vframe(child));
            }
            weak_child = GC_MALLOC_ATOMIC(sizeof(void *));
            *weak_child = nested_children[1].u.p;
            assert(GC_general_register_disappearing_link(weak_child, nested_children[1].u.p) == GC_SUCCESS);
            gx_yield_task(t);
            return;
        }
        assert(registrations == 2);
        gx_cancel_ctx(context, gx_context_canceled());
        assert(!"fault did not escape");
    }
    if (mode == 18) {
        if (f->pc++ == 0) {
            timer_callbacks = 0;
            gx_add_timer_at(1, NULL, timer_order, gx_int(1));
            gx_add_timer_at(0, NULL, timer_order, gx_int(2));
            gx_add_timer_at(1, NULL, timer_order, gx_int(3));
            gx_yield_task(t);
            return;
        }
        assert(timer_callbacks == 3 && order[0] == 2 && order[1] == 1 && order[2] == 3);
        gx_ret(t, f);
        return;
    }
    if (mode == 13) {
        if (f->pc++ == 0) {
            gx_V payload = gx_str((const char *)binary, sizeof(binary));
            weak_due = GC_MALLOC_ATOMIC(sizeof(void *));
            *weak_due = payload.u.p;
            assert(GC_general_register_disappearing_link(weak_due, payload.u.p) == GC_SUCCESS);
            timer_callbacks = 0;
            gx_add_timer_at(0, NULL, timer_first, gx_nil());
            gx_add_timer_at(0, NULL, timer_second, payload);
            gx_yield_task(t);
            return;
        }
        assert(timer_callbacks == 2 && gx_sched->ntimers == 0);
        gx_ret(t, f);
        return;
    }
    if (mode == 14) {
        for (int i = 0; i < 512; i++) {
            gx_V parent = gx_at(gx_std_context_with_timeout(gx_background(), gx_int(INT64_MAX)), 0);
            gx_V child = gx_at(gx_std_context_with_timeout(parent, gx_int(INT64_MAX)), 0);
            assert(((gx_Context *)child.u.p)->deadline == ((gx_Context *)parent.u.p)->deadline);
            gx_cancel_ctx(parent, gx_context_canceled());
            assert(!gx_sched->contexts && gx_sched->ntimers == 0);
            assert(!((gx_Context *)parent.u.p)->children && !((gx_Context *)child.u.p)->parent);
        }
        gx_V expired = gx_at(gx_std_context_with_timeout(gx_background(), gx_int(-1)), 0);
        gx_V child = gx_at(gx_std_context_with_timeout(expired, gx_int(5)), 0);
        assert(((gx_Context *)expired.u.p)->err.u.p == gx_context_deadline_exceeded().u.p);
        assert(((gx_Context *)child.u.p)->err.u.p == gx_context_deadline_exceeded().u.p);
        assert(!gx_sched->contexts && gx_sched->ntimers == 0);
        assert(((gx_Context *)gx_background().u.p)->n == 0);
        gx_ret(t, f);
        return;
    }
    if (mode >= 20) {
        if (f->pc++ == 0) {
            context = gx_at(gx_std_context_with_cancel(gx_background()), 0);
            retained_channel = gx_make_chan(gx_int(0), gx_zero_nil);
            retained_mutex = gx_new_mutex();
            gx_std_sync_mutex_lock(t, retained_mutex);
            retained_waitgroup = gx_new_waitgroup();
            gx_std_sync_waitgroup_add(retained_waitgroup, gx_int(1));
            gx_spawn(gx_vframe(gx_new_frame(0, blocked_step, NULL)));
            gx_yield_task(t);
            return;
        }
        assert(registrations == 1);
        if (retained_waiter) {
            gx_Task *task = retained_waiter->task;
            gx_V *rv = task->rv;
            gx_Panic *panic = task->resume_panic;
            bool selected = retained_waiter->sel ? *retained_waiter->sel : false;
            pthread_t thread;
            assert(!GC_pthread_create(&thread, NULL, wrong_thread_waiter, NULL));
            assert(!GC_pthread_join(thread, NULL));
            assert(retained_waiter->task == task && task->rv == rv && task->resume_panic == panic);
            assert(!retained_waiter->sel || *retained_waiter->sel == selected);
            assert(((gx_Context *)context.u.p)->err.t == GX_NIL);
        }
        gx_ret(t, f);
        return;
    }
    if (mode == 2 || mode == 7 || mode == 16) {
        int count = mode == 7 ? 256 : 3;
        if (f->pc == 0) {
            f->pc = 1;
            for (int i = 0; i < count; i++)
                gx_spawn(gx_vframe(gx_new_frame(0, child_step, NULL)));
            gx_yield_task(t);
            return;
        }
        if (f->pc == 1) {
            assert(registrations == count);
            if (mode == 16) {
                for (int i = 0; i < count; i++) {
                    gx_host_publish(tokens[i], binary, sizeof(binary), i == 0 ? "first native fault" : NULL);
                    gx_host_ack(tokens[i]);
                    gx_host_token_release(tokens[i]);
                }
                gx_yield_task(t);
                return;
            }
            if (mode == 7) {
                for (int i = count - 1; i >= 0; i--) {
                    gx_host_publish(tokens[i], binary, sizeof(binary), NULL);
                    gx_host_ack(tokens[i]);
                    gx_host_token_release(tokens[i]);
                }
                f->pc = 4;
                gx_yield_task(t);
                return;
            }
            gx_host_publish(tokens[0], binary, sizeof(binary), NULL);
            gx_host_publish(tokens[1], binary, sizeof(binary), NULL);
            gx_host_ack(tokens[1]);
            f->pc = 2;
            gx_yield_task(t);
            return;
        }
        if (f->pc == 2) {
            assert(decoded == 1 && order[0] == 2);
            gx_host_publish(tokens[2], binary, sizeof(binary), NULL);
            gx_host_ack(tokens[2]);
            f->pc = 3;
            gx_yield_task(t);
            return;
        }
        if (f->pc == 3) {
            assert(decoded == 2 && order[1] == 3);
            gx_host_ack(tokens[0]);
            f->pc = 4;
            gx_yield_task(t);
            return;
        }
        if (decoded != count || gx_sched->all != t) {
            gx_yield_task(t);
            return;
        }
        assert(cleaned == count && gx_sched->pending == NULL && gx_sched->ntimers == 0);
        if (mode == 2) {
            assert(order[2] == 1);
            for (int i = 0; i < 3; i++)
                gx_host_token_release(tokens[i]);
        } else
            for (int i = 0; i < count; i++)
                assert(order[i] == count - i);
        gx_ret(t, f);
        return;
    }
    if (f->pc++ == 0) {
        context = gx_at(gx_std_context_with_cancel(gx_background()), 0);
        gx_HostBoundary boundary = gx_host_boundary(context, 0);
        if (mode == 8)
            boundary = gx_host_boundary(gx_nil(), 0);
        if (mode == 9)
            boundary = gx_host_boundary(saved_context, 0);
        if (mode == 10) {
            boundary = gx_host_boundary(context, 1000000);
            while (gx_now() < boundary.deadline) {
            } /* explicit absolute-deadline gate before submit */
        }
        gx_HostAction cancel = mode == 6 ? cancel_fault : cancel_count;
        gx_HostAction cleanup = mode == 11 ? cleanup_fault : cleanup_count;
        if (mode == 1 || mode == 12 || mode == 15) {
            NativeJob *j = calloc(1, sizeof(*j));
            native_job = j;
            pthread_mutex_init(&j->mutex, NULL);
            pthread_cond_init(&j->gate, NULL);
            uint8_t producer[4];
            memcpy(producer, binary, sizeof(producer));
            j->input = malloc(sizeof(binary));
            memcpy(j->input, producer, sizeof(binary));
            memset(producer, 7, sizeof(producer));
            j->token = gx_host_register(t, boundary, gx_str((const char *)binary, sizeof(binary)), decode_bytes,
                                        native_cancel, native_cleanup, j);
            gx_host_token_retain(j->token);
            assert(!GC_pthread_create(&j->thread, NULL, native_work, j));
            tokens[0] = j->token;
            if (mode == 12) {
                t->blocked = false;
                gx_ret(t, f);
            } /* main return cancels and waits actual ACK */
            else
                gx_spawn(gx_vframe(gx_new_frame(0, mode == 15 ? busy_step : progress_step, NULL)));
            return;
        }
        gx_HostToken token = gx_host_register(t, boundary, gx_str((const char *)binary, sizeof(binary)), decode_bytes,
                                              cancel, cleanup, NULL);
        tokens[0] = token;
        gx_HostToken wrong = token;
        wrong.task++;
        assert(!gx_host_publish(wrong, NULL, 0, "wrong task fault"));
        assert(!gx_host_ack(wrong));
        wrong = token;
        wrong.generation++;
        assert(!gx_host_publish(wrong, binary, sizeof(binary), NULL));
        assert(!gx_host_ack(wrong));
        if (gx_host_should_submit(token)) {
            uint8_t result[4];
            memcpy(result, binary, sizeof(result));
            assert(gx_host_publish(token, result, sizeof(result), mode == 4 ? "native worker failure" : NULL));
            memset(result, 42, sizeof(result));
            assert(!gx_host_publish(token, binary, sizeof(binary), "duplicate fault"));
            wrong = token;
            wrong.task++;
            assert(!gx_host_ack(wrong));
            assert(gx_host_live_count(token) == 1);
            if (mode == 0)
                gx_spawn(gx_vframe(gx_new_frame(0, ack_barrier_step, NULL)));
            else {
                assert(gx_host_ack(token));
                assert(!gx_host_ack(token));
            }
        }
        if (mode == 3 || mode == 4 || mode == 6 || mode == 17)
            gx_cancel_ctx(context, gx_context_canceled());
        return;
    }
    if (mode == 15) {
        if (gx_sched->all != t) {
            gx_yield_task(t);
            return;
        }
        assert(busy_progress > 0);
    }
    assert(decoded == 1 && cleaned == 1);
    if (mode == 0)
        saved_result = t->rv[0];
    gx_cancel_ctx(context, gx_context_canceled());
    assert(!gx_host_publish(tokens[0], binary, sizeof(binary), NULL));
    assert(!gx_host_ack(tokens[0]));
    assert(gx_host_live_count(tokens[0]) == 0);
    gx_ret(t, f);
}
static gx_V entry(void) { return gx_vframe(gx_new_frame(0, root_step, NULL)); }
static void init(void) {
    decoded = cleaned = canceled = registrations = step_progress = busy_progress = 0;
    atomic_store(&native_started, false);
    retained_waiter = NULL;
    native_job = NULL;
    if (mode == 19)
        gx_host_fault("entry constructor fault");
    assert(gx_deadline(INT64_MAX - 3, 5) == INT64_MAX);
    assert(gx_deadline(9, -1) == 9);
}
int main(void) {
    GC_INIT();
    for (mode = 0; mode <= 19; mode++) {
        if (mode == 9)
            continue; /* run foreign context after mode0 separately */
        int status = gx_run_main_host(init, entry);
        assert(status ==
               ((mode == 4 || mode == 5 || mode == 6 || mode == 11 || mode == 16 || mode == 17 || mode == 19) ? 3 : 0));
        assert(cleaned == (mode == 13 || mode == 14 || mode == 18 || mode == 19 ? 0
                           : mode == 2 || mode == 16                            ? 3
                           : mode == 7                                          ? 256
                                                                                : 1));
        assert(atomic_load(&leases) == 0);
        if (mode != 2 && mode != 7 && mode != 16 && mode != 13 && mode != 14 && mode != 18 && mode != 19) {
            assert(!gx_host_ack(tokens[0]));
            assert(!gx_host_publish(tokens[0], binary, sizeof(binary), NULL));
            gx_host_token_release(tokens[0]);
        }
        assert(gx_sched->retired && !gx_sched->all && !gx_sched->contexts && !gx_sched->pending && !gx_sched->timers &&
               !gx_sched->runq);
        if (mode == 0) {
            saved_context = context;
            int saved_mode = mode;
            mode = 9;
            assert(gx_run_main_host(init, entry) == 0);
            assert(decoded == 1 && cleaned == 1);
            gx_host_token_release(tokens[0]);
            mode = saved_mode;
        }
    }
    assert(saved_result.l == sizeof(binary) && !memcmp(gx_sbytes(saved_result), binary, sizeof(binary)));
    mode = 30;
    assert(gx_run_main_host(init, entry) == 3);
    assert(canceled == 2 && cleaned == 2);
    for (int i = 0; i < 2; i++) {
        gx_Context *c = nested_children[i].u.p;
        assert(c->err.u.p == gx_context_canceled().u.p && !c->parent && !c->children);
        gx_host_token_release(tokens[i]);
    }
    for (mode = 20; mode <= 27; mode++) {
        assert(gx_run_main_host(init, entry) == 0);
        gx_Chan *c = retained_channel.u.p;
        assert(!c->sendq.head && !c->recvq.head);
        assert(((gx_Mutex *)retained_mutex.u.p)->n == 0);
        assert(((gx_WaitGroup *)retained_waitgroup.u.p)->nw == 0);
        if (retained_waiter) {
            assert(!retained_waiter->task && retained_waiter->val.t == GX_NIL);
            gx_send_done(retained_waiter, true);
            gx_recv_done(retained_waiter, gx_nil(), false);
        }
        gx_cancel_ctx(context, gx_nil());
        assert(!gx_sched->all && !gx_sched->contexts);
    }
    harness_constructor_failure();
    puts("C host identity/FIFO/snapshots/ACK/decoder-fault/cancel/256-results/blocked-retirement passed");
    return 0;
}
