// std.time.sleep: suspend for d nanoseconds of virtual time.
import { park, sched, yieldTask } from "./task_spawn.ts";

export function* stdTimeSleep(d: bigint): Generator<unknown, void, unknown> {
  if (d <= 0n) {
    yield* yieldTask();
    return;
  }
  sched.addTimer(d, sched.cur, null);
  yield* park();
}
