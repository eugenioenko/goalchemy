// core.task.spawn and the cooperative scheduler. Each source task is a
// generator; exactly one runs at a time and runnable tasks are dispatched in
// FIFO order. A task runs until it blocks, yields, returns, or panics.
import { GoPanic } from "../types/panic.ts";
import { formatChain, isGen, panicState, type PanicState } from "../types/program.ts";
import { writeStderr } from "../types/print.ts";

type Gen = Generator<unknown, unknown, unknown>;

export class Task implements PanicState {
  id: number;
  gen: Gen;
  done = false;
  curPanic: GoPanic | null = null;
  deferTarget: unknown = undefined;
  constructor(id: number, gen: Gen) {
    this.id = id;
    this.gen = gen;
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

export class Scheduler {
  runq: Task[] = [];
  cur: Task;
  nextId = 1;
  rng: number;
  clock = 0n;
  timers: Timer[] = [];
  seq = 0;
  onDeadlock: (() => void) | null = null;
  constructor(cur: Task) {
    this.cur = cur;
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
  addTimer(d: bigint, task: Task | null, fn: (() => void) | null): void {
    this.seq++;
    this.timers.push({ at: this.clock + d, seq: this.seq, task, fn });
  }
  /** Next runnable task, advancing virtual time when only timers remain. */
  next(): Task {
    while (this.runq.length === 0) {
      if (this.timers.length === 0) {
        if (this.onDeadlock !== null) this.onDeadlock();
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
}

function seed(): number {
  const s = Number(process.env.GOALCHEMY_SEED ?? "1");
  return Number.isInteger(s) && s > 0 && s < 2 ** 32 ? s : 1;
}

export function fatal(msg: string): never {
  writeStderr("fatal error: " + msg + "\n");
  process.exit(2);
}

export let sched: Scheduler = new Scheduler(new Task(0, (function* () {})()));

/** The value a generator yields to give control back to the scheduler. */
export const SUSPEND = Symbol("suspend");

/** Suspends the running task until something readies it. */
export function* park(): Generator<unknown, void, unknown> {
  yield SUSPEND;
}

function runTask(t: Task): void {
  sched.cur = t;
  const r = t.gen.next();
  if (r.done) t.done = true;
}

/** Runs the program entry as task 0 and drives every task to completion of
 * the entry; unrecovered panics in any task terminate the program. */
export function runMain(entry: () => Gen): void {
  const main = new Task(0, entry());
  sched = new Scheduler(main);
  panicState.current = () => sched.cur;
  sched.ready(main);
  for (;;) {
    const t = sched.next();
    try {
      runTask(t);
    } catch (e) {
      if (e instanceof GoPanic) {
        writeStderr(formatChain(e));
        process.exit(2);
      }
      if (e instanceof RangeError && /call stack/.test(e.message)) {
        writeStderr("runtime: goroutine stack exceeds limit\nfatal error: stack overflow\n");
        process.exit(2);
      }
      throw e;
    }
    if (main.done) return;
  }
}

/** Runs one harness case in a fresh scheduler; throws Blocked on deadlock. */
export function runIsolated(entry: () => Gen): unknown {
  const main = new Task(0, entry());
  sched = new Scheduler(main);
  sched.onDeadlock = () => {
    throw new Blocked();
  };
  panicState.current = () => sched.cur;
  sched.ready(main);
  let result: unknown;
  const wrapped = main.gen;
  main.gen = (function* () {
    result = yield* wrapped;
  })();
  for (;;) {
    runTask(sched.next());
    if (main.done) return result;
  }
}

/** go f(args): the callee and arguments are already evaluated. */
export function spawn(f: ((...a: any[]) => any) | null, args: any[]): void {
  const gen: Gen =
    f === null
      ? (function* () {
          throw new GoPanic(null);
        })()
      : isGen(f)
        ? f(...args)
        : (function* () {
            f(...args);
          })();
  const t = new Task(sched.nextId++, gen);
  sched.ready(t);
}

/** runtime.Gosched for the scheduler: requeue the running task. */
export function* yieldTask(): Generator<unknown, void, unknown> {
  sched.ready(sched.cur);
  yield SUSPEND;
}
