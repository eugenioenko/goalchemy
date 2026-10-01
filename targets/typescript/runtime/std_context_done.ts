// std.context.done: the channel closed on cancellation.
import type { Chan } from "./chan_make.ts";
import type { Context } from "./std_context_err.ts";

export function stdContextContextDone(c: Context): Chan | null {
  return c.done;
}
