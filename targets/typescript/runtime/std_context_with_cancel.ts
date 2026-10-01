// std.context.with_cancel: a child cancelled by its cancel function.
import { cancelContext, CONTEXT_CANCELED, newChild, type Context } from "./std_context_err.ts";

export function stdContextWithCancel(parent: Context): [Context, () => void] {
  const c = newChild(parent);
  return [c, () => cancelContext(c, CONTEXT_CANCELED)];
}
