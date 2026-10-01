// std.sync.mutex.lock: acquire or wait FIFO; Unlock hands the lock over.
import { park, sched, type Task } from "./task_spawn.ts";

export class Mutex {
  locked = false;
  waiters: Task[] = [];
  $clone(): Mutex {
    const m = new Mutex();
    m.locked = this.locked;
    m.waiters = this.waiters.slice();
    return m;
  }
  $set(o: Mutex): void {
    this.locked = o.locked;
    this.waiters = o.waiters.slice();
  }
}

export function* stdSyncMutexLock(m: Mutex): Generator<unknown, void, unknown> {
  if (!m.locked) {
    m.locked = true;
    return;
  }
  m.waiters.push(sched.cur);
  yield* park();
}
