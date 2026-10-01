// std.time.sleep: suspend for d nanoseconds of virtual time; a pause
// primitive.
import { sched, yieldTask, type Task } from "./task_spawn.ts";

export function stdTimeSleep(t: Task, d: bigint): void {
  t.rv = [];
  if (d <= 0n) {
    yieldTask(t);
    return;
  }
  sched.addTimer(d, t, null);
  sched.block(t);
}
