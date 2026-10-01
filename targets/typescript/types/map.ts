// Go maps: insertion-ordered entries with snapshot iteration.

export class Entry<K, V> {
  k: K;
  v: V;
  live = true;
  constructor(k: K, v: V) {
    this.k = k;
    this.v = v;
  }
}

export class GoMap<K, V> {
  index = new Map<unknown, Entry<K, V>>();
  entries: Entry<K, V>[] = [];
  readonly keyOf: (k: K) => unknown;
  constructor(keyOf: (k: K) => unknown) {
    this.keyOf = keyOf;
  }
  compact(): void {
    if (this.entries.length > 32 && this.entries.length > 2 * this.index.size) {
      this.entries = this.entries.filter((e) => e.live);
    }
  }
}

export class MapIter<K, V> {
  entries: Entry<K, V>[];
  i = 0;
  k: K | undefined;
  v: V | undefined;
  constructor(entries: Entry<K, V>[]) {
    this.entries = entries;
  }
}

/** Key encoding for key types whose host values compare by value. */
export function identityKey<K>(k: K): unknown {
  return k;
}
