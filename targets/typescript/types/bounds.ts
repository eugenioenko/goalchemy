// Go slice-bounds checks and messages.
import { runtimePanic } from "./panic.ts";

type N = number | bigint;

function oob(msg: string): never {
  throw runtimePanic("slice bounds out of range " + msg);
}

/** Checks lo:hi against a length or capacity limit named by word. */
export function check2(lo: N, hi: N, limit: number, word: string): void {
  if (hi < 0) oob(`[:${hi}]`);
  if (hi > limit) oob(`[:${hi}] with ${word} ${limit}`);
  if (lo < 0) oob(`[${lo}:]`);
  if (lo > hi) oob(`[${lo}:${hi}]`);
}

/** Checks lo:hi:max against a capacity limit named by word. */
export function check3(lo: N, hi: N, max: N, limit: number, word: string): void {
  if (max < 0) oob(`[::${max}]`);
  if (max > limit) oob(`[::${max}] with ${word} ${limit}`);
  if (hi < 0) oob(`[:${hi}:]`);
  if (hi > max) oob(`[:${hi}:${max}]`);
  if (lo < 0) oob(`[${lo}::]`);
  if (lo > hi) oob(`[${lo}:${hi}:]`);
}
