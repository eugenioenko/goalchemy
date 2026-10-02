import "../../types/node_host.ts";
// Serves runtime conformance requests for the TypeScript target over JSON
// Lines on standard input and output.
import { createInterface } from "node:readline";
import { writeSync } from "node:fs";
import { cases } from "./harness_gen.ts";
import { GoPanic } from "../../types/panic.ts";
import { formatPanicValue } from "../../types/program.ts";
import { Blocked, resetScheduler } from "../../runtime/task_spawn.ts";

const PROTOCOL = 1;

function respond(r: unknown): void {
  writeSync(1, JSON.stringify(r) + "\n");
}

function serve(req: any): unknown {
  const base = { v: PROTOCOL, id: req.id };
  if (req.v !== PROTOCOL) return { ...base, status: "harness_failure", error: "unsupported protocol version" };
  const fn = cases[req.case];
  if (fn === undefined) return { ...base, status: "harness_failure", error: "unknown case " + req.case };
  const after: Record<string, unknown> = {};
  const h = { let: (n: string) => req.let[n], after: (n: string, v: unknown) => { after[n] = v; } };
  try {
    resetScheduler();
    const results = fn(h);
    return { ...base, status: "returned", results, after };
  } catch (e) {
    if (e instanceof Blocked) return { ...base, status: "blocked" };
    if (e instanceof GoPanic) {
      return { ...base, status: "panic", panic: Buffer.from(formatPanicValue(e.value), "latin1").toString("utf8") };
    }
    return { ...base, status: "harness_failure", error: String(e) };
  }
}

const rl = createInterface({ input: process.stdin });
rl.on("line", (line) => {
  let req: any;
  try {
    req = JSON.parse(line);
  } catch (e) {
    respond({ v: PROTOCOL, status: "harness_failure", error: String(e) });
    return;
  }
  respond(serve(req));
});
