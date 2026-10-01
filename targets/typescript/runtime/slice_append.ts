// core.slice.append: append with Goalchemy's growth rule. Aggregate elements
// are copied with clone when they move to new storage.
import { NIL, Slice } from "../types/slice.ts";
import { fault } from "../types/panic.ts";

const MAX_CAP = Number.MAX_SAFE_INTEGER;

export function growCap(old: number, required: number): number {
  const doubled = old <= MAX_CAP / 2 ? 2 * old : MAX_CAP;
  return Math.max(required, Math.max(1, doubled));
}

function appendValues<T>(s: Slice<T>, vs: T[], clone?: (v: T) => T): Slice<T> {
  const n = s.l + vs.length;
  if (vs.length === 0) return s;
  if (n <= s.c) {
    const a = s.a!;
    for (let i = 0; i < vs.length; i++) a[s.o + s.l + i] = vs[i];
    return new Slice(a, s.o, n, s.c);
  }
  const c = growCap(s.c, n);
  if (c > 2 ** 32 - 1) throw fault("slice growth to " + c + " elements exceeds host limits");
  const a = new Array<T>(n);
  for (let i = 0; i < s.l; i++) {
    const v = s.a![s.o + i];
    a[i] = clone ? clone(v) : v;
  }
  for (let i = 0; i < vs.length; i++) a[s.l + i] = vs[i];
  return new Slice(a, 0, n, c);
}

/** append(s, e1, e2, ...) with already-copied element values. */
export function append<T>(s: Slice<T>, vs: T[], clone?: (v: T) => T): Slice<T> {
  return appendValues(s, vs, clone);
}

/** append(s, t...), copying t first so overlapping storage is safe. */
export function appendSlice<T>(s: Slice<T>, t: Slice<T>, clone?: (v: T) => T): Slice<T> {
  if (t.l === 0) return s.a === null && t.a === null ? NIL : s;
  const vs = new Array<T>(t.l);
  for (let i = 0; i < t.l; i++) {
    const v = t.a![t.o + i];
    vs[i] = clone ? clone(v) : v;
  }
  return appendValues(s, vs, clone);
}

/** append(b, s...) for a byte slice and a string. */
export function appendString(b: Slice<number>, s: string): Slice<number> {
  const vs = new Array<number>(s.length);
  for (let i = 0; i < s.length; i++) vs[i] = s.charCodeAt(i);
  return appendValues(b, vs);
}
