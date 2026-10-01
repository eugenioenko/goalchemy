// std.sync.waitgroup.wait: return at zero, otherwise wait.
import type { WaitGroup } from "./std_sync_waitgroup_add.ts";
import { park, sched } from "./task_spawn.ts";

export function* stdSyncWaitgroupWait(wg: WaitGroup): Generator<unknown, void, unknown> {
  if (wg.n === 0n) return;
  wg.waiters.push(sched.cur);
  yield* park();
}
