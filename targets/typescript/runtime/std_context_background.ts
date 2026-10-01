// std.context.background: the never-cancelled root context.
import { BACKGROUND, type Context } from "./std_context_err.ts";

export function stdContextBackground(): Context {
  return BACKGROUND;
}
