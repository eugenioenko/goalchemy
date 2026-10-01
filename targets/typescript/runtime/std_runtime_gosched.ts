// std.runtime.gosched: requeue the running task; a pause primitive.
import { yieldTask, type Task } from "./task_spawn.ts";

export function stdRuntimeGosched(t: Task): void {
  t.rv = [];
  yieldTask(t);
}
