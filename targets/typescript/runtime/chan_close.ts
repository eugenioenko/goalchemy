// core.chan.close: close(ch), releasing waiting receivers and senders.
import { dequeue, type Chan } from "./chan_make.ts";
import { plainPanic } from "../types/panic.ts";

export function chanClose(ch: Chan | null): void {
  if (ch === null) throw plainPanic("close of nil channel");
  if (ch.closed) throw plainPanic("close of closed channel");
  ch.closed = true;
  for (let w = dequeue(ch.recvq); w !== null; w = dequeue(ch.recvq)) w.complete(undefined, false, false);
  for (let w = dequeue(ch.sendq); w !== null; w = dequeue(ch.sendq)) w.complete(undefined, false, true);
}
