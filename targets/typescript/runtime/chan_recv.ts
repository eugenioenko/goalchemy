// core.chan.recv: v, ok := <-ch, a pause primitive; results arrive in
// t.rv as [value, ok].
import { tryRecv, parkWaiter, Waiter, type Chan } from "./chan_make.ts";
import { sched, type Task } from "./task_spawn.ts";

export function chanRecv(t: Task, ch: Chan | null): void {
  if (ch === null) {
    sched.block(t);
    return;
  }
  const [v, ok, done] = tryRecv(ch);
  if (done) {
    t.rv = [v, ok];
    return;
  }
  parkWaiter(t, ch.recvq, new Waiter(t, undefined));
}
