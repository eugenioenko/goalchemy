// core.chan.cap: buffer capacity.
import type { Chan } from "./chan_make.ts";

export function chanCap(ch: Chan | null): bigint {
  return ch === null ? 0n : BigInt(ch.size);
}
