// core.chan.len: buffered element count.
import type { Chan } from "./chan_make.ts";

export function chanLen(ch: Chan | null): bigint {
  return ch === null ? 0n : BigInt(ch.buf.length);
}
