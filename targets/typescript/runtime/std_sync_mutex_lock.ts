// std.sync.mutex.lock: acquire or wait FIFO; Unlock hands the lock over.
import { sched, type Task } from "./task_spawn.ts";

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

/** A pause primitive. */
export function stdSyncMutexLock(t: Task, m: Mutex): void {
  t.rv = [];
  if (!m.locked) {
    m.locked = true;
    return;
  }
  m.waiters.push(t);
  t.cleanup = () => { m.waiters = m.waiters.filter(w => w !== t); };
  sched.block(t);
}
