// Pointers to non-aggregate storage. Pointers to structs and arrays are the
// stable host objects themselves.

export interface Ref<T> {
  v: T;
}

/** Heap storage for a boxed variable or new(T). */
export class Cell<T> implements Ref<T> {
  v: T;
  constructor(v: T) {
    this.v = v;
  }
}

/** A pointer to a struct field holding a non-aggregate value. */
export class FieldRef<T> implements Ref<T> {
  o: any;
  k: string;
  constructor(o: any, k: string) {
    this.o = o;
    this.k = k;
  }
  get v(): T {
    return this.o[this.k];
  }
  set v(x: T) {
    this.o[this.k] = x;
  }
}

const fieldRefs = new WeakMap<object, Map<string, FieldRef<any>>>();

/** Returns the canonical pointer to o's field k, so pointer equality holds. */
export function fieldRef<T>(o: object, k: string): FieldRef<T> {
  let m = fieldRefs.get(o);
  if (m === undefined) {
    m = new Map();
    fieldRefs.set(o, m);
  }
  let r = m.get(k);
  if (r === undefined) {
    r = new FieldRef<T>(o, k);
    m.set(k, r);
  }
  return r;
}
