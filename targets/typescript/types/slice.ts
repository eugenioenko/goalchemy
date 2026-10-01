// Slice headers over shared backing arrays. Headers are immutable values.

export class Slice<T> {
  readonly a: T[] | null;
  readonly o: number;
  readonly l: number;
  readonly c: number;
  constructor(a: T[] | null, o: number, l: number, c: number) {
    this.a = a;
    this.o = o;
    this.l = l;
    this.c = c;
  }
}

/** The nil slice. Headers are immutable, so one instance serves all types. */
export const NIL: Slice<any> = new Slice<any>(null, 0, 0, 0);

export function isNil(s: Slice<unknown>): boolean {
  return s.a === null;
}

/** Wraps a host array as a slice with len == cap. */
export function fromArray<T>(a: T[]): Slice<T> {
  return new Slice(a, 0, a.length, a.length);
}

export function toArray<T>(s: Slice<T>): T[] {
  return s.a === null ? [] : s.a.slice(s.o, s.o + s.l);
}

export function sliceLen(s: Slice<unknown>): number {
  return s.l;
}
