// Portable host boundary. Executables install the Node adapter; browsers supply
// output/failure explicitly. Wall-clock lib.clock capabilities are separate.
export interface RuntimeHost {
  now(): number; // monotonic milliseconds
  alarm(ms: number, wake: () => void): () => void;
  stdout(bytes: Uint8Array): void;
  stderr(bytes: Uint8Array): void;
  fail(code: number): never;
  seed: number;
}
export const portableHost: RuntimeHost = {
  now: () => performance.now(),
  alarm: (ms, wake) => {
    const id = setTimeout(wake, ms);
    return () => clearTimeout(id);
  },
  stdout: () => { throw new Error("runtime stdout adapter required"); },
  stderr: () => { throw new Error("runtime stderr adapter required"); },
  fail: (code) => { throw new Error("runtime executable exit " + code); },
  seed: 1,
};
export let runtimeHost = portableHost;
export function installRuntimeHost(host: RuntimeHost): void { runtimeHost = host; }
export function binaryBytes(s: string): Uint8Array {
  return Uint8Array.from(s, c => c.charCodeAt(0) & 255);
}
