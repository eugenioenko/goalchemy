// Timeout is anchored at creation and inherits the earliest absolute deadline.
import { cancelContext, CONTEXT_CANCELED, CONTEXT_DEADLINE_EXCEEDED, newChild, type Context } from "./std_context_err.ts";
import { sched } from "./task_spawn.ts";
export function stdContextWithTimeout(parent: Context, d: bigint): [Context, () => void] {
  const c = newChild(parent);
  if (c.err === null) {
    const now = sched.now();
    const sum = now + (d > 0n ? d : 0n);
    const at = sum > 9223372036854775807n ? 9223372036854775807n : sum;
    c.deadline = c.deadline === null || at < c.deadline ? at : c.deadline;
    if (c.deadline <= now) cancelContext(c, CONTEXT_DEADLINE_EXCEEDED);
    else c.stopTimer = sched.addTimerAt(c.deadline, null, () => cancelContext(c, CONTEXT_DEADLINE_EXCEEDED));
  }
  return [c, () => cancelContext(c, CONTEXT_CANCELED)];
}
