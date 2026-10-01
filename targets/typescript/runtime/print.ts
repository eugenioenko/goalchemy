// core.print: print and println of Booleans, integers, and strings.
import { writeStderr } from "../types/print.ts";

export function printString(v: boolean | number | bigint | string): string {
  return typeof v === "string" ? v : String(v);
}

export function print(args: (boolean | number | bigint | string)[], newline: boolean): void {
  let out = "";
  for (let i = 0; i < args.length; i++) {
    if (newline && i > 0) out += " ";
    out += printString(args[i]);
  }
  if (newline) out += "\n";
  writeStderr(out);
}
