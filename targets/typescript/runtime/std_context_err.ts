// std.context.err and the context representation: a cancellation state with
// a Done channel closed on cancellation; Background's channel is nil.
import type { Box } from "../types/iface.ts";
import type { Chan } from "./chan_make.ts";
import { makeChan } from "./chan_make.ts";
import { chanClose } from "./chan_close.ts";
import { stdErrorsNew } from "./std_errors_new.ts";

export class Context {
  done: Chan | null;
  err: Box | null = null;
  children: Context[] = [];
  constructor(done: Chan | null) {
    this.done = done;
  }
}

export const CONTEXT_CANCELED = stdErrorsNew("context canceled");
export const CONTEXT_DEADLINE_EXCEEDED = stdErrorsNew("context deadline exceeded");
export const BACKGROUND = new Context(null);

export function cancelContext(c: Context, err: Box): void {
  if (c.err !== null) return;
  c.err = err;
  chanClose(c.done);
  for (const k of c.children) cancelContext(k, err);
  c.children = [];
}

export function newChild(parent: Context): Context {
  const c = new Context(makeChan(0));
  if (parent.err !== null) cancelContext(c, parent.err);
  else if (parent !== BACKGROUND) parent.children.push(c);
  return c;
}

export function stdContextContextErr(c: Context): Box | null {
  return c.err;
}
