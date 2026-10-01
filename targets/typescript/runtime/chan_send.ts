// core.chan.send: ch <- v.
import { dequeue, Waiter, type Chan } from "./chan_make.ts";
import { plainPanic } from "../types/panic.ts";
import { park, sched } from "./task_spawn.ts";

export function* chanSend(ch: Chan | null, v: unknown): Generator<unknown, void, unknown> {
  if (ch === null) {
    yield* park();
    return;
  }
  if (ch.closed) throw plainPanic("send on closed channel");
  const r = dequeue(ch.recvq);
  if (r !== null) {
    r.complete(v, true, false);
    return;
  }
  if (ch.buf.length < ch.size) {
    ch.buf.push(v);
    return;
  }
  const w = new Waiter(sched.cur, v);
  ch.sendq.push(w);
  yield* park();
  if (w.closed) throw plainPanic("send on closed channel");
}
