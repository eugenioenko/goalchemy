// Program-level runtime: deferred calls, recover, closures, and the entry
// point that reports unrecovered panics as the Go runtime does.
import { box, type Box, type TypeDesc, typeDesc } from "./iface.ts";
import { GoPanic, nilDeref, runtimePanic, TYPE_ASSERTION_ERROR } from "./panic.ts";
import { writeStderr } from "./print.ts";
import { runtimeHost } from "./host.ts";
import { floatPrint } from "./float.ts";

export interface Deferred {
  f: ((...args: any[]) => any) | null;
  args: any[];
  fid: unknown;
  /** Set when f starts a frame (a suspending callee). */
  start?: boolean;
}

/** Per-task recover state: the panic visible to recover and the identity of
 * the function the defer machinery invoked directly. */
export interface PanicState {
  curPanic: GoPanic | null;
  deferTarget: unknown;
}

const mainState: PanicState = { curPanic: null, deferTarget: undefined };

/** Returns the running task's recover state; the scheduler replaces it. */
export const panicState = { current: (): PanicState => mainState };
export function resetPanicBinding(): void {
  mainState.curPanic = null; mainState.deferTarget = undefined;
  panicState.current = () => mainState;
}

const PANIC_NIL_ERROR: TypeDesc = typeDesc({
  name: "*runtime.PanicNilError",
  kind: "runtime_error",
  eq: (a, b) => a === b,
  key: (a) => String(a),
  methods: { Error: () => "runtime error: panic called with nil argument", RuntimeError: () => {} },
});

/** panic(v): a nil interface becomes *runtime.PanicNilError as in Go 1.21+. */
export function goPanic(v: Box | null): GoPanic {
  return new GoPanic(v === null ? box(PANIC_NIL_ERROR, {}) : v);
}

/** Converts a caught host exception into the source panic it carries. */
export function catchPanic(e: unknown): GoPanic {
  if (e instanceof GoPanic) return e;
  if (e instanceof RangeError && /call stack/.test(e.message)) {
    throw e;
  }
  throw e;
}

/** Runs deferred calls in reverse order, implementing recover and re-panics. */
export function runDefers(ds: Deferred[], p: GoPanic | null): void {
  let panicking = p;
  while (ds.length > 0) {
    const d = ds.pop()!;
    const st = panicState.current();
    const savedPanic = st.curPanic;
    const savedTarget = st.deferTarget;
    st.curPanic = panicking;
    st.deferTarget = d.fid;
    try {
      if (d.f === null) throw nilDeref();
      d.f(...d.args);
    } catch (e) {
      const np = catchPanic(e);
      if (np !== panicking && np.prev === null) np.prev = panicking;
      panicking = np;
      continue;
    } finally {
      st.curPanic = savedPanic;
      st.deferTarget = savedTarget;
    }
    if (panicking !== null && panicking.recovered) panicking = null;
  }
  if (panicking !== null) throw panicking;
}

/** recover() called from the function identified by fid. */
export function recover(fid: unknown): Box | null {
  const st = panicState.current();
  const p = st.curPanic;
  if (p === null || p.recovered || st.deferTarget !== fid) return null;
  p.recovered = true;
  return p.value;
}

type Fn = ((...args: any[]) => any) & { $fid?: unknown };

/** Tags a function value with the identity recover compares against. */
export function closure<F extends Fn>(fid: unknown, f: F): F {
  f.$fid = fid;
  return f;
}

export function bound(fid: unknown, f: Fn, recv: any): Fn {
  return closure(fid, (...a: any[]) => f(recv, ...a));
}

export function ichk(x: Box | null): Box {
  if (x === null) throw nilDeref();
  return x;
}

/** Method value from an interface: evaluates the receiver now. */
export function ibound(x: Box | null, id: string): Fn {
  const b = ichk(x);
  const m = b.t.methods[id] as Fn;
  return closure(m.$fid, (...a: any[]) => m(b.v, ...a));
}

export function fnchk<F>(f: F | null): F {
  if (f === null) throw nilDeref();
  return f;
}

export function fid(f: Fn | null): unknown {
  return f === null ? undefined : f.$fid;
}

/** Builds the error for a failed non-comma-ok type assertion. */
export function assertPanic(x: Box | null, iface: string, target: string, missing: string | null): GoPanic {
  let msg: string;
  if (x === null) msg = "interface conversion: " + iface + " is nil, not " + target;
  else if (missing !== null) msg = "interface conversion: " + x.t.name + " is not " + target + ": missing method " + missing;
  else msg = "interface conversion: " + iface + " is " + x.t.name + ", not " + target;
  return new GoPanic(box(TYPE_ASSERTION_ERROR, msg));
}

function indented(s: string): string {
  return s.replace(/\n/g, "\n\t");
}

/** Renders a panic value as runtime.printpanicval does. */
export function formatPanicValue(v: Box | null): string {
  if (v === null) return "nil";
  const m = v.t.methods;
  if (typeof m["Error"] === "function") return indented(String(m["Error"](v.v)));
  if (typeof m["String"] === "function") return indented(String(m["String"](v.v)));
  const builtin = !v.t.name.includes(".");
  switch (v.t.basic) {
    case "string":
      return builtin ? indented(v.v) : v.t.name + '("' + indented(v.v) + '")';
    case "float32":
    case "float64": {
      const text = floatPrint(v.v, v.t.basic === "float32" ? 32 : 64);
      return builtin ? text : v.t.name + "(" + text + ")";
    }
    case "int":
    case "bool":
      return builtin ? String(v.v) : v.t.name + "(" + String(v.v) + ")";
  }
  return "(" + v.t.name + ") 0xc000000000";
}

export function formatChain(p: GoPanic): string {
  let s = "";
  if (p.prev !== null) s += formatChain(p.prev) + "\t";
  s += "panic: " + formatPanicValue(p.value);
  if (p.recovered) s += " [recovered]";
  return s + "\n";
}

/** Runs the program entry point; unrecovered panics exit with status 2. */
export function main(entry: () => void): void {
  try {
    entry();
  } catch (e) {
    if (e instanceof GoPanic) {
      writeStderr(formatChain(e));
      runtimeHost.fail(2);
    }
    if (e instanceof RangeError && /call stack/.test(e.message)) {
      writeStderr("runtime: goroutine stack exceeds limit\nfatal error: stack overflow\n");
      runtimeHost.fail(2);
    }
    throw e;
  }
}

export { runtimePanic };

/** The type descriptor for plain string panic values raised by runtimes. */
export const STRING_TYPE: TypeDesc = typeDesc({
  name: "string",
  kind: "string",
  eq: (a, b) => a === b,
  key: (a) => a,
  methods: {},
  basic: "string",
});
