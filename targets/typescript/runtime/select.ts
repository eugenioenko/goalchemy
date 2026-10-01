// core.select: commit exactly one ready case.
import { dequeue, hasLive, SelectState, tryRecv, Waiter, type Chan } from "./chan_make.ts";
import { plainPanic } from "../types/panic.ts";
import { park, sched } from "./task_spawn.ts";

export interface SelectCase {
  ch: Chan | null;
  send: boolean;
  val?: unknown;
}

/** Returns [index, value, ok]; index is -1 for the default case. */
export function* select(cases: SelectCase[], hasDefault: boolean): Generator<unknown, [number, unknown, boolean], unknown> {
  const ready: number[] = [];
  cases.forEach((c, i) => {
    const ch = c.ch;
    if (ch === null) return;
    if (c.send) {
      if (ch.closed || hasLive(ch.recvq) || ch.buf.length < ch.size) ready.push(i);
    } else if (ch.buf.length > 0 || hasLive(ch.sendq) || ch.closed) {
      ready.push(i);
    }
  });
  if (ready.length > 0) {
    const i = ready[sched.choose(ready.length)];
    const c = cases[i];
    const ch = c.ch!;
    if (c.send) {
      if (ch.closed) throw plainPanic("send on closed channel");
      const w = dequeue(ch.recvq);
      if (w !== null) w.complete(c.val, true, false);
      else ch.buf.push(c.val);
      return [i, undefined, false];
    }
    const [v, ok] = tryRecv(ch);
    return [i, v, ok];
  }
  if (hasDefault) return [-1, undefined, false];
  const st = new SelectState();
  const ws: (Waiter | null)[] = cases.map((c, i) => {
    if (c.ch === null) return null;
    const w = new Waiter(sched.cur, c.send ? c.val : undefined, st, i);
    (c.send ? c.ch.sendq : c.ch.recvq).push(w);
    return w;
  });
  yield* park();
  cases.forEach((c, i) => {
    if (c.ch !== null) {
      c.ch.sendq = c.ch.sendq.filter((w) => w !== ws[i]);
      c.ch.recvq = c.ch.recvq.filter((w) => w !== ws[i]);
    }
  });
  const w = ws[st.index]!;
  if (cases[st.index].send && w.closed) throw plainPanic("send on closed channel");
  return [st.index, w.val, w.ok];
}
