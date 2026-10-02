// core.chan.send: ch <- v, a pause primitive.
import { dequeue, parkWaiter, Waiter, type Chan } from "./chan_make.ts";
import { plainPanic } from "../types/panic.ts";
import { sched, type Task } from "./task_spawn.ts";

export function chanSend(t: Task, ch: Chan | null, v: unknown): void {
  t.rv = [];
  if (ch === null) {
    sched.block(t);
    return;
  }
  if (ch.closed) throw plainPanic("send on closed channel");
  const r = dequeue(ch.recvq);
  if (r !== null) {
    r.recvDone(v, true);
    return;
  }
  if (ch.buf.length < ch.size) {
    ch.buf.push(v);
    return;
  }
  parkWaiter(t, ch.sendq, new Waiter(t, v));
}
