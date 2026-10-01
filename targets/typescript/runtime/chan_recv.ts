// core.chan.recv: v, ok := <-ch.
import { tryRecv, Waiter, type Chan } from "./chan_make.ts";
import { park, sched } from "./task_spawn.ts";

export function* chanRecv(ch: Chan | null, zero: () => unknown): Generator<unknown, [unknown, boolean], unknown> {
  if (ch === null) {
    yield* park();
    return [zero(), false];
  }
  const [v, ok, done] = tryRecv(ch);
  if (done) return ok ? [v, true] : [zero(), false];
  const w = new Waiter(sched.cur, undefined);
  ch.recvq.push(w);
  yield* park();
  return w.ok ? [w.val, true] : [zero(), false];
}
