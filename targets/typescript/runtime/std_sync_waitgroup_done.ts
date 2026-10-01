// std.sync.waitgroup.done: Add(-1).
import { stdSyncWaitgroupAdd, type WaitGroup } from "./std_sync_waitgroup_add.ts";

export function stdSyncWaitgroupDone(wg: WaitGroup): void {
  stdSyncWaitgroupAdd(wg, -1n);
}
