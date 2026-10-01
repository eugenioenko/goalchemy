// Byte-exact output to standard error and standard output.
import { writeSync } from "node:fs";

export function writeStderr(s: string): void {
  writeSync(2, Buffer.from(s, "latin1"));
}

export function writeStdout(s: string): void {
  writeSync(1, Buffer.from(s, "latin1"));
}
