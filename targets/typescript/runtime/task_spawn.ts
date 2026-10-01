// core.task.spawn and the cooperative scheduler. Suspending functions are
// compiled to resumable frames: a frame holds the function's locals and the
// block to resume at, and step runs it until it returns or reaches a pause
// point. A task is a stack of frames driven by a trampoline; exactly one
// task runs at a time and runnable tasks are dispatched in FIFO order.
// Pause primitives either complete immediately, leaving their results in
// task.rv, or block the task until another task or a timer readies it.
// Deferred calls, panics, and recover are managed per task by the runtime.
import { GoPanic, nilDeref } from "../types/panic.ts";
import { catchPanic, formatChain, panicState, type Deferred, type PanicState } from "../types/program.ts";
import { writeStderr } from "../types/print.ts";
import { Slice } from "../types/slice.ts";

export abstract class Frame {
  pc = 0;
  defers: Deferred[] = [];
  parent: Frame | null = null;
  panicking: GoPanic | null = null;
  abstract step(t: Task): void;
  results(): unknown[] {
    return [];
  }
}

export class Task implements PanicState {
  id: number;
  frame: Frame | null;
  rv: unknown[] = [];
  blocked = false;
  done = false;
  resumePanic: GoPanic | null = null;
  curPanic: GoPanic | null = null;
  deferTarget: unknown = undefined;
  cleanup: (() => void) | null = null;
  constructor(id: number, frame: Frame) {
    this.id = id;
    this.frame = frame;
  }
}

interface Timer {
  at: bigint;
  seq: number;
  task: Task | null;
  fn: (() => void) | null;
}

/** Thrown out of a harness case when its task blocks with nothing runnable. */
export class Blocked extends Error {}

class FatalPanic extends Error {
  p: GoPanic;
  constructor(p: GoPanic) {
    super("fatal panic");
    this.p = p;
  }
}

export class Scheduler {
  runq: Task[] = [];
  cur: Task;
  main: Task;
  nextId = 1;
  rng: number;
  clock = 0n;
  timers: Timer[] = [];
  seq = 0;
  harness = false;
  constructor(main: Task) {
    this.cur = main;
    this.main = main;
    this.rng = seed();
  }
  /** xorshift32 choice source, identical on every target. */
  choose(n: number): number {
    let x = this.rng;
    x ^= x << 13;
    x >>>= 0;
    x ^= x >>> 17;
    x ^= x << 5;
    x >>>= 0;
    this.rng = x;
    return x % n;
  }
  ready(t: Task): void {
    this.runq.push(t);
  }
  block(t: Task): void {
    t.blocked = true;
  }
  addTimer(d: bigint, task: Task | null, fn: (() => void) | null): void {
    this.seq++;
    this.timers.push({ at: this.clock + d, seq: this.seq, task, fn });
  }
  next(): Task {
    while (this.runq.length === 0) {
      if (this.timers.length === 0) {
        if (this.harness) throw new Blocked();
        fatal("all goroutines are asleep - deadlock!");
      }
      this.fireTimers();
    }
    return this.runq.shift()!;
  }
  fireTimers(): void {
    let at = this.timers[0].at;
    for (const t of this.timers) if (t.at < at) at = t.at;
    this.clock = at;
    const due = this.timers.filter((t) => t.at === at).sort((a, b) => a.seq - b.seq);
    this.timers = this.timers.filter((t) => t.at !== at);
    for (const t of due) {
      if (t.fn !== null) t.fn();
      if (t.task !== null) this.ready(t.task);
    }
  }
  /** Drives a task's frames until it blocks or finishes. */
  run(t: Task): void {
    this.cur = t;
    t.blocked = false;
    if (t.cleanup !== null) {
      const c = t.cleanup;
      t.cleanup = null;
      c();
    }
    while (!t.blocked && t.frame !== null) {
      const p = t.resumePanic;
      if (p !== null) {
        t.resumePanic = null;
        this.exit(t, t.frame, p);
        continue;
      }
      const f = t.frame;
      try {
        f.step(t);
      } catch (e) {
        this.exit(t, f, catchPanic(e));
      }
    }
  }
  /** Leaves frame f (panicking when p is set); deferred calls run first. */
  exit(t: Task, f: Frame, p: GoPanic | null): void {
    if (p !== null) {
      if (p.prev === null && f.panicking !== null && f.panicking !== p) p.prev = f.panicking;
      f.panicking = p;
    }
    t.frame = f;
    if (f.defers.length > 0) {
      const r = new DeferRunner(f);
      r.parent = f;
      t.frame = r;
      return;
    }
    this.finish(t, f);
  }
  /** Pops f and delivers its results, or its panic, to the caller. */
  finish(t: Task, f: Frame): void {
    const p = f.panicking;
    const parent = f.parent;
    t.frame = parent;
    if (parent === null) {
      t.done = true;
      if (p !== null) {
        if (this.harness) throw new FatalPanic(p);
        writeStderr(formatChain(p));
        process.exit(2);
      }
      return;
    }
    if (parent instanceof DeferRunner && parent.child === f) {
      parent.childDone(t, p);
      return;
    }
    if (p !== null) {
      this.exit(t, parent, p);
      return;
    }
    t.rv = f.results();
  }
}

/** Runs a frame's deferred calls in reverse order. */
class DeferRunner extends Frame {
  target: Frame;
  child: Frame | null = null;
  savedPanic: GoPanic | null = null;
  savedTarget: unknown = undefined;
  constructor(target: Frame) {
    super();
    this.target = target;
  }
  step(t: Task): void {
    const tf = this.target;
    while (tf.defers.length > 0) {
      const d = tf.defers.pop()!;
      this.savedPanic = t.curPanic;
      this.savedTarget = t.deferTarget;
      t.curPanic = tf.panicking;
      t.deferTarget = d.fid;
      if (d.start) {
        let child: Frame;
        try {
          if (d.f === null) throw nilDeref();
          child = d.f(...d.args);
        } catch (e) {
          t.curPanic = this.savedPanic;
          t.deferTarget = this.savedTarget;
          this.after(tf, catchPanic(e));
          continue;
        }
        this.child = child;
        child.parent = this;
        t.frame = child;
        return;
      }
      let p: GoPanic | null = null;
      try {
        if (d.f === null) throw nilDeref();
        d.f(...d.args);
      } catch (e) {
        p = catchPanic(e);
      }
      t.curPanic = this.savedPanic;
      t.deferTarget = this.savedTarget;
      this.after(tf, p);
    }
    t.frame = tf;
    sched.finish(t, tf);
  }
  childDone(t: Task, p: GoPanic | null): void {
    t.curPanic = this.savedPanic;
    t.deferTarget = this.savedTarget;
    this.child = null;
    t.frame = this;
    this.after(this.target, p);
  }
  after(tf: Frame, p: GoPanic | null): void {
    if (p !== null) {
      if (p.prev === null && tf.panicking !== null && tf.panicking !== p) p.prev = tf.panicking;
      tf.panicking = p;
      return;
    }
    if (tf.panicking !== null && tf.panicking.recovered) tf.panicking = null;
  }
}

function seed(): number {
  const s = Number(process.env.GOALCHEMY_SEED ?? "1");
  return Number.isInteger(s) && s > 0 && s < 2 ** 32 ? s : 1;
}

export function fatal(msg: string): never {
  writeStderr("fatal error: " + msg + "\n");
  process.exit(2);
}

export let sched: Scheduler = new Scheduler(new Task(0, null as unknown as Frame));

/** Pushes a callee frame; the caller resumes with its results in t.rv. */
export function call(t: Task, child: Frame): void {
  child.parent = t.frame;
  t.frame = child;
}

/** Returns from frame f. */
export function ret(t: Task, f: Frame): void {
  sched.exit(t, f, null);
}

/** Runs an ordinary call as a frame. */
class SyncFrame extends Frame {
  fn: () => unknown[];
  res: unknown[] = [];
  constructor(fn: () => unknown[]) {
    super();
    this.fn = fn;
  }
  step(t: Task): void {
    this.res = this.fn();
    ret(t, this);
  }
  results(): unknown[] {
    return this.res;
  }
}

export function sync(fn: () => unknown[]): Frame {
  return new SyncFrame(fn);
}

type Fn = ((...args: any[]) => any) & { $fid?: unknown };

/** Adapts an ordinary function value with n results to the resumable form. */
export function adapt(f: Fn | null, n: number): Fn | null {
  if (f === null) return null;
  const g: Fn = (...a: any[]) =>
    sync(() => {
      const r = f(...a);
      return n === 0 ? [] : n === 1 ? [r] : r;
    });
  g.$fid = f.$fid;
  return g;
}

/** Adapts each ordinary function in a slice to the resumable form. */
export function adaptSlice(s: Slice<Fn | null>, n: number): Slice<Fn | null> {
  if (s.a === null) return s;
  const a: (Fn | null)[] = [];
  for (let i = 0; i < s.l; i++) a.push(adapt(s.a[s.o + i], n));
  return new Slice(a, 0, a.length, a.length);
}

/** go f(args): starts a task running frame f. */
export function spawn(f: Frame): void {
  const t = new Task(sched.nextId++, f);
  sched.ready(t);
}

function install(main: Task, harness: boolean): Scheduler {
  sched = new Scheduler(main);
  sched.harness = harness;
  panicState.current = () => sched.cur;
  return sched;
}

/** Runs the program entry as the first task until it returns. */
export function runMain(entry: Frame): void {
  const main = new Task(0, entry);
  const s = install(main, false);
  s.ready(main);
  while (!main.done) {
    try {
      s.run(s.next());
    } catch (e) {
      if (e instanceof RangeError && /call stack/.test(e.message)) {
        writeStderr("runtime: goroutine stack exceeds limit\nfatal error: stack overflow\n");
        process.exit(2);
      }
      throw e;
    }
  }
}

/** Requeues the running task: a pause primitive. */
export function yieldTask(t: Task): void {
  sched.ready(t);
  sched.block(t);
}

class AwaitFrame extends Frame {
  fn: (t: Task) => void;
  res: unknown[] = [];
  constructor(fn: (t: Task) => void) {
    super();
    this.fn = fn;
  }
  step(t: Task): void {
    if (this.pc === 0) {
      this.pc = 1;
      this.fn(t);
      return;
    }
    this.res = t.rv;
    ret(t, this);
  }
  results(): unknown[] {
    return this.res;
  }
}

/** Runs one pause primitive in an isolated scheduler for a harness case;
 * throws Blocked when no task can run, and the source panic on panic. */
export function runIsolated(fn: (t: Task) => void): unknown[] {
  const h = new AwaitFrame(fn);
  const main = new Task(0, h);
  const s = install(main, true);
  s.ready(main);
  try {
    while (!main.done) s.run(s.next());
  } catch (e) {
    if (e instanceof FatalPanic) throw e.p;
    throw e;
  }
  return h.res;
}

/** Installs an isolated scheduler for harness cases without pauses. */
export function resetScheduler(): void {
  install(new Task(0, null as unknown as Frame), true);
}
