// std.runtime.gosched: requeue the running task.
import { yieldTask } from "./task_spawn.ts";

export function* stdRuntimeGosched(): Generator<unknown, void, unknown> {
  yield* yieldTask();
}
