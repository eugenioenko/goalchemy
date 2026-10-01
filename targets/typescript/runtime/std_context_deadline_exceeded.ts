// std.context.deadline_exceeded: the context.DeadlineExceeded sentinel.
import type { Box } from "../types/iface.ts";
import { CONTEXT_DEADLINE_EXCEEDED } from "./std_context_err.ts";

export function stdContextDeadlineExceeded(): Box {
  return CONTEXT_DEADLINE_EXCEEDED;
}
