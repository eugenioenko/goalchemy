// Byte-exact output through the explicit host adapter.
import { binaryBytes, runtimeHost } from "./host.ts";
export function writeStderr(s: string): void { runtimeHost.stderr(binaryBytes(s)); }
export function writeStdout(s: string): void { runtimeHost.stdout(binaryBytes(s)); }
