// std.sync.waitgroup.wait: return at zero, otherwise wait; a pause primitive.
import type { WaitGroup } from "./std_sync_waitgroup_add.ts";
import { sched, type Task } from "./task_spawn.ts";

export function stdSyncWaitgroupWait(t: Task, wg: WaitGroup): void {
  t.rv = [];
  if (wg.n === 0n) return;
  wg.waiters.push(t);
  t.cleanup = () => { wg.waiters = wg.waiters.filter(w => w !== t); };
  sched.block(t);
}
