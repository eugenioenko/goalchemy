// std.context.with_timeout: a child cancelled after a virtual-time duration.
import { cancelContext, CONTEXT_CANCELED, CONTEXT_DEADLINE_EXCEEDED, newChild, type Context } from "./std_context_err.ts";
import { sched } from "./task_spawn.ts";

export function stdContextWithTimeout(parent: Context, d: bigint): [Context, () => void] {
  const c = newChild(parent);
  if (c.err === null) sched.addTimer(d, null, () => cancelContext(c, CONTEXT_DEADLINE_EXCEEDED));
  return [c, () => cancelContext(c, CONTEXT_CANCELED)];
}
