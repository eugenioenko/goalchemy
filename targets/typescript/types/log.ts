// Host log sink for std/log/slog. Levels follow log/slog: -4 debug, 0 info,
// 4 warn, 8 error. Without a handler, records at warn and above go to stderr.
import { binaryBytes, runtimeHost } from "./host.ts";
export interface LogRecord {
  level: number;
  unixNano: bigint;
  time: Date;
  message: string;
  attrs: [string, string][];
  text: string;
}
export type LogHandler = (record: LogRecord) => void;
let logHandler: LogHandler | null = null;
let logLevel = 4;
export function setLogHandler(handler: LogHandler | null, level: number = 4): void {
  if (!Number.isSafeInteger(level)) throw new TypeError("log level must be an integer");
  logHandler = handler; logLevel = level;
}
export function logLevelNumber(level: bigint): number {
  return level < -0x20000000000000n ? -0x20000000000000 : level > 0x20000000000000n ? 0x20000000000000 : Number(level);
}
export function logEnabled(level: number): boolean { return level >= logLevel; }
const utf8 = new TextDecoder();
function text(s: string): string { return utf8.decode(binaryBytes(s)); }
export function logEmit(level: number, unixNano: bigint, message: string, attrs: string[], line: string): void {
  const handler = logHandler;
  if (handler === null) {
    try { runtimeHost.stderr(binaryBytes(line + "\n")); } catch { console.error(text(line)); }
    return;
  }
  const pairs: [string, string][] = [];
  for (let i = 0; i + 1 < attrs.length; i += 2) pairs.push([text(attrs[i]!), text(attrs[i + 1]!)]);
  try {
    handler({ level, unixNano, time: new Date(Number(unixNano / 1000000n)), message: text(message), attrs: pairs, text: text(line) });
  } catch {}
}
