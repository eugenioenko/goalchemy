// std.context.canceled: the context.Canceled sentinel.
import type { Box } from "../types/iface.ts";
import { CONTEXT_CANCELED } from "./std_context_err.ts";

export function stdContextCanceled(): Box {
  return CONTEXT_CANCELED;
}
