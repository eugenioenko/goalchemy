// Slice headers over shared backing arrays. Headers are immutable values.

/** Indexed storage shared by headers; Uint8Array stores Go uint8 values directly. */
export interface Backing<T> {
  [i: number]: T;
  readonly length: number;
  [Symbol.iterator](): IterableIterator<T>;
  slice(start?: number, end?: number): Backing<T>;
}

export function allocate<T>(n: number, bytes: boolean): Backing<T> {
  return bytes ? new Uint8Array(n) as unknown as Backing<T> : new Array<T>(n);
}

export class Slice<T> {
  readonly a: Backing<T> | null;
  readonly bytes: boolean;
  readonly o: number;
  readonly l: number;
  readonly c: number;
  constructor(a: Backing<T> | null, o: number, l: number, c: number, bytes = a instanceof Uint8Array) {
    this.bytes = bytes;
    this.a = a;
    this.o = o;
    this.l = l;
    this.c = c;
  }
}

/** Generic nil slice; the byte hint selects native storage on later append. */
export const NIL: Slice<any> = new Slice<any>(null, 0, 0, 0);

export const BYTE_NIL: Slice<number> = new Slice<number>(null, 0, 0, 0, true);

export function isNil(s: Slice<unknown>): boolean {
  return s.a === null;
}

/** Wraps a host array as a slice with len == cap. */
export function fromArray<T>(a: Backing<T>): Slice<T> {
  return new Slice(a, 0, a.length, a.length);
}

export function toArray<T>(s: Slice<T>): T[] {
  const a: T[] = [];
  for (let i = 0; i < s.l; i++) a.push(s.a![s.o + i]);
  return a;
}

export function sliceLen(s: Slice<unknown>): number {
  return s.l;
}

/** Initialize newly allocated spare capacity without overwriting reused backing. */
export function zeroAppendGrowth<T>(result:Slice<T>, previous:Slice<T>, zero:()=>T):Slice<T> {
  if (result.a !== previous.a && !result.bytes) for(let i=result.l;i<result.c;i++) result.a![i]=zero();
  return result;
}
