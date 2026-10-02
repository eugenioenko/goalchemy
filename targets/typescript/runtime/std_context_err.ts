// Context state belongs to its creating driver. Native callbacks never mutate it.
import type { Box } from "../types/iface.ts";
import type { Chan } from "./chan_make.ts";
import { makeChan } from "./chan_make.ts";
import { chanClose } from "./chan_close.ts";
import { stdErrorsNew } from "./std_errors_new.ts";
import { nilDeref } from "../types/panic.ts";
import { HostFault, sched, type Scheduler } from "./task_spawn.ts";
export class Context {
  err: Box | null = null;
  children = new Set<Context>();
  hooks = new Set<() => void>();
  parent: Context | null = null;
  deadline: bigint | null = null;
  release: (() => void) | null = null;
  stopTimer: (() => void) | null = null;
  done: Chan | null;
  owner: Scheduler | null;
  constructor(done: Chan | null, owner: Scheduler | null = null) { this.done=done; this.owner=owner; }
}
export const CONTEXT_CANCELED = stdErrorsNew("context canceled");
export const CONTEXT_DEADLINE_EXCEEDED = stdErrorsNew("context deadline exceeded");
export const BACKGROUND = new Context(null);
export function cancelContext(c: Context, err: Box): void {
  if (c === BACKGROUND || c.err !== null) return;
  c.err = err;
  c.release?.(); c.release = null;
  c.stopTimer?.(); c.stopTimer = null;
  c.parent?.children.delete(c); c.parent = null;
  chanClose(c.done);
  let failure: unknown;
  for (const k of c.children) { try { cancelContext(k, err); } catch (e) { failure ??= e; } }
  c.children.clear();
  const hooks = [...c.hooks]; c.hooks.clear();
  for (const h of hooks) { try { h(); } catch (e) { failure ??= e; } }
  if (failure !== undefined) throw new HostFault("context cancel hook: " + String(failure));
}
export function observeContext(c: Context): void {
  if (c === null) throw nilDeref();
  if (c.owner !== null && c.owner !== sched) throw new HostFault("foreign context owner");
  if (c.err === null && c.deadline !== null && c.deadline <= sched.now()) cancelContext(c, CONTEXT_DEADLINE_EXCEEDED);
}
export function newChild(parent: Context): Context {
  observeContext(parent);
  const c = new Context(makeChan(0, () => ({})), sched);
  const dispose = () => {
    c.stopTimer?.(); c.stopTimer = null;
    c.parent?.children.delete(c); c.parent = null;
    c.children.clear(); c.hooks.clear(); c.release = null;
  };
  sched.disposers.add(dispose);
  const owner = sched;
  c.release = () => { owner.disposers.delete(dispose); };
  c.deadline = parent.deadline;
  if (parent.err !== null) cancelContext(c, parent.err);
  else if (parent !== BACKGROUND) { parent.children.add(c); c.parent = parent; }
  return c;
}
export function onContextCancel(c: Context, hook: () => void): () => void {
  observeContext(c);
  if (c === BACKGROUND) return () => {};
  if (c.err !== null) { hook(); return () => {}; }
  c.hooks.add(hook);
  return () => { c.hooks.delete(hook); };
}
export function stdContextContextErr(c: Context): Box | null { observeContext(c); return c.err; }
