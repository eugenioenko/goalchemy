// core.chan.make: make(chan T, size), and the channel state shared by the
// channel and select runtime functions.
import { GoPanic, plainPanic } from "../types/panic.ts";
import { sched, type Task } from "./task_spawn.ts";

export class SelectState {
  done = false;
}

export class Waiter {
  task: Task;
  val: unknown;
  sel: SelectState | null;
  idx: number;
  constructor(task: Task, val: unknown, sel: SelectState | null = null, idx = 0) {
    this.task = task;
    this.val = val;
    this.sel = sel;
    this.idx = idx;
  }
  /** Completes a waiting receiver with a value or closure. */
  recvDone(val: unknown, ok: boolean): void {
    if (this.sel !== null) {
      this.sel.done = true;
      this.task.rv = [this.idx, val, ok];
    } else {
      this.task.rv = [val, ok];
    }
    sched.ready(this.task);
  }
  /** Completes a waiting sender; closed makes it panic when it resumes. */
  sendDone(closed: boolean): void {
    if (this.sel !== null) {
      this.sel.done = true;
      this.task.rv = [this.idx, undefined, false];
    } else {
      this.task.rv = [];
    }
    if (closed) this.task.resumePanic = plainPanic("send on closed channel") as GoPanic;
    sched.ready(this.task);
  }
}

export class Chan {
  buf: unknown[] = [];
  size: number;
  closed = false;
  recvq: Waiter[] = [];
  sendq: Waiter[] = [];
  zero: () => unknown;
  constructor(size: number, zero: () => unknown) {
    this.size = size;
    this.zero = zero;
  }
}

/** Removes and returns the first waiter that can still complete. */
export function dequeue(q: Waiter[]): Waiter | null {
  while (q.length > 0) {
    const w = q.shift()!;
    if (w.sel === null || !w.sel.done) return w;
  }
  return null;
}

export function hasLive(q: Waiter[]): boolean {
  return q.some((w) => w.sel === null || !w.sel.done);
}

/** Receives without blocking: [value, ok, done]. */
export function tryRecv(ch: Chan): [unknown, boolean, boolean] {
  if (ch.buf.length > 0) {
    const v = ch.buf.shift();
    const w = dequeue(ch.sendq);
    if (w !== null) {
      ch.buf.push(w.val);
      w.sendDone(false);
    }
    return [v, true, true];
  }
  const w = dequeue(ch.sendq);
  if (w !== null) {
    const v = w.val;
    w.sendDone(false);
    return [v, true, true];
  }
  if (ch.closed) return [ch.zero(), false, true];
  return [undefined, false, false];
}

export function makeChan(size: number | bigint, zero: () => unknown = () => undefined): Chan {
  if (size < 0 || size > Number.MAX_SAFE_INTEGER) throw plainPanic("makechan: size out of range");
  return new Chan(Number(size), zero);
}
