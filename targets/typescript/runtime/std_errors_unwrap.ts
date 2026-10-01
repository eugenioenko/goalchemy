// std.errors.unwrap: the result of an Unwrap() error method, or nil.
import type { Box } from "../types/iface.ts";

export function stdErrorsUnwrap(err: Box | null): Box | null {
  if (err === null) return null;
  const unwrap = err.t.methods["Unwrap"];
  return unwrap === undefined ? null : unwrap(err.v);
}
