// std.errors.is: errors.Is over the Unwrap() error chain.
import type { Box } from "../types/iface.ts";

export function stdErrorsIs(err: Box | null, target: Box | null): boolean {
  if (err === null || target === null) return err === target;
  const comparable = target.t.comparable !== false;
  let cur: Box | null = err;
  while (cur !== null) {
    if (comparable && cur.t === target.t && cur.t.eq(cur.v, target.v)) return true;
    const is = cur.t.methods["Is"];
    if (is !== undefined && is(cur.v, target)) return true;
    const unwrap: ((v: unknown) => Box | null) | undefined = cur.t.methods["Unwrap"];
    if (unwrap === undefined) return false;
    cur = unwrap(cur.v);
  }
  return false;
}
