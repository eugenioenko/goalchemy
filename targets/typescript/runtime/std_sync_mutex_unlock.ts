// std.sync.mutex.unlock: hand the lock to the first waiter or release it.
import type { Mutex } from "./std_sync_mutex_lock.ts";
import { fatal, sched } from "./task_spawn.ts";

export function stdSyncMutexUnlock(m: Mutex): void {
  if (!m.locked) fatal("sync: unlock of unlocked mutex");
  const t = m.waiters.shift();
  if (t !== undefined) {
    sched.ready(t);
    return;
  }
  m.locked = false;
}
