// Canonical value encoding for the TypeScript harness.
import { Slice, NIL } from "../../types/slice.ts";
import { GoMap, identityKey } from "../../types/map.ts";
import { mapSet } from "../../runtime/map_store.ts";
import { mapIter, mapNext } from "../../runtime/map_iterate.ts";
import { fromHost } from "../../types/utf8.ts";
import { stdErrorsNew } from "../../runtime/std_errors_new.ts";
import type { Box } from "../../types/iface.ts";

export function decError(raw: any): Box | null {
  return raw.nil ? null : stdErrorsNew(fromHost(raw.error));
}

export function encError(v: Box | null): unknown {
  if (v === null) return { nil: true };
  return { error: Buffer.from(String(v.t.methods["Error"](v.v)), "latin1").toString("utf8") };
}

import * as rt from "./runtime_index.ts";

export function decChan<T>(raw: any, dec: (r: any) => T): rt.Chan | null {
  if (raw.nil) return null;
  const ch = rt.makeChan(Number(raw.cap ?? "0"));
  for (const v of raw.chan) ch.buf.push(dec(v));
  if (raw.closed) ch.closed = true;
  return ch;
}

export function encChan<T>(ch: rt.Chan | null, enc: (v: T) => unknown): unknown {
  if (ch === null) return { nil: true };
  return { chan: ch.buf.map((v) => enc(v as T)), cap: String(ch.size), closed: ch.closed };
}

export function encZero(_: unknown): unknown {
  return { zero: true };
}

export function* harnessSpawn(): Generator<unknown, boolean, unknown> {
  let ran = false;
  rt.spawn(() => {
    ran = true;
  }, []);
  const before = ran;
  yield* rt.yieldTask();
  if (!ran) throw new Error("spawned task did not run after the parent yielded");
  return before;
}

export function* harnessSelect2(a: rt.Chan | null, b: rt.Chan | null, dflt: boolean): Generator<unknown, bigint, unknown> {
  const [i] = yield* rt.select([{ ch: a, send: false }, { ch: b, send: false }], dflt);
  return BigInt(i);
}

export function* harnessLockUnlock(m: rt.Mutex): Generator<unknown, void, unknown> {
  yield* rt.stdSyncMutexLock(m);
  rt.stdSyncMutexUnlock(m);
}

export interface H {
  let(name: string): any;
  after(name: string, v: unknown): void;
}

const WIDE = new Set(["i64", "u64"]);

export function decInt(raw: string, kind: string): number | bigint {
  return WIDE.has(kind) ? BigInt(raw) : Number(raw);
}

export function decBool(raw: string): boolean {
  return raw === "true";
}

export function decString(raw: any): string {
  if (typeof raw.str === "string") return fromHost(raw.str);
  return Buffer.from(raw.hex, "hex").toString("latin1");
}

export function decSlice<T>(raw: any, dec: (r: any) => T): Slice<T> {
  if (raw.nil) return NIL;
  const items: T[] = raw.slice.map(dec);
  const c = raw.cap !== undefined ? Number(raw.cap) : items.length;
  const a = new Array<T>(c);
  for (let i = 0; i < items.length; i++) a[i] = items[i];
  for (let i = items.length; i < c; i++) a[i] = items.length > 0 ? items[0] : (undefined as T);
  return new Slice(a, 0, items.length, c);
}

export function view<T>(base: Slice<T>, raw: any): Slice<T> {
  const lo = raw.lo !== undefined ? Number(raw.lo) : 0;
  const hi = raw.hi !== undefined ? Number(raw.hi) : base.l;
  const max = raw.max !== undefined ? Number(raw.max) : base.c;
  return new Slice(base.a, base.o + lo, hi - lo, max - lo);
}

export function decMap<K, V>(raw: any, dk: (r: any) => K, dv: (r: any) => V): GoMap<K, V> | null {
  if (raw.nil) return null;
  const m = new GoMap<K, V>(identityKey);
  for (const e of raw.map) mapSet(m, dk(e.key), dv(e.value));
  return m;
}

export function encInt(v: any): string {
  return String(v);
}

export function encBool(v: any): string {
  return v ? "true" : "false";
}

export function encString(v: any): unknown {
  return { hex: Buffer.from(v, "latin1").toString("hex") };
}

export function encSlice<T>(s: Slice<T>, enc: (v: T) => unknown): unknown {
  if (s.a === null) return { nil: true };
  const items: unknown[] = [];
  for (let i = 0; i < s.l; i++) items.push(enc(s.a[s.o + i]));
  return { slice: items, cap: String(s.c) };
}

export function encArray<T>(a: T[], enc: (v: T) => unknown): unknown {
  return { array: a.map(enc) };
}

export function encMap<K, V>(m: GoMap<K, V> | null, ek: (k: K) => unknown, ev: (v: V) => unknown): unknown {
  if (m === null) return { nil: true };
  const items: unknown[] = [];
  const it = mapIter(m);
  while (mapNext(it)) items.push({ key: ek(it.k as K), value: ev(it.v as V) });
  return { map: items };
}
