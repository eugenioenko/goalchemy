// core.slice.slice: s[lo:hi:max] sharing backing storage.
import { NIL, Slice } from "../types/slice.ts";
import { check2, check3 } from "../types/bounds.ts";

type N = number | bigint;

export function reslice<T>(s: Slice<T>, lo?: N, hi?: N, max?: N): Slice<T> {
  const l = lo ?? 0;
  const h = hi ?? s.l;
  let m: N = s.c;
  if (max !== undefined) {
    check3(l, h, max, s.c, "capacity");
    m = max;
  } else {
    check2(l, h, s.c, "capacity");
  }
  if (s.a === null) return NIL;
  const ln = Number(l);
  return new Slice(s.a, s.o + ln, Number(h) - ln, Number(m) - ln);
}

/** Slices an array through a pointer: (&a)[lo:hi:max]. */
export function sliceArray<T>(a: T[], lo?: N, hi?: N, max?: N): Slice<T> {
  const l = lo ?? 0;
  const h = hi ?? a.length;
  let m: N = a.length;
  if (max !== undefined) {
    check3(l, h, max, a.length, "length");
    m = max;
  } else {
    check2(l, h, a.length, "length");
  }
  const ln = Number(l);
  return new Slice(a, ln, Number(h) - ln, Number(m) - ln);
}
