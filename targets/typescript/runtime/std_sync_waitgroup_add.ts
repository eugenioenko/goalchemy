// std.sync.waitgroup.add: adjust the counter; release waiters at zero.
import { box } from "../types/iface.ts";
import { GoPanic } from "../types/panic.ts";
import { sched, type Task } from "./task_spawn.ts";
import { STRING_TYPE } from "../types/program.ts";

export class WaitGroup {
  n = 0n;
  waiters: Task[] = [];
  $clone(): WaitGroup {
    const w = new WaitGroup();
    w.n = this.n;
    w.waiters = this.waiters.slice();
    return w;
  }
  $set(o: WaitGroup): void {
    this.n = o.n;
    this.waiters = o.waiters.slice();
  }
}

export function stdSyncWaitgroupAdd(wg: WaitGroup, d: bigint): void {
  wg.n += d;
  if (wg.n < 0n) throw new GoPanic(box(STRING_TYPE, "sync: negative WaitGroup counter"));
  if (wg.n === 0n) {
    for (const t of wg.waiters) sched.ready(t);
    wg.waiters = [];
  }
}
