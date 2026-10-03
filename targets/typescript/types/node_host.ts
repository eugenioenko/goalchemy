// Node executable adapter. This module is excluded from portable runtime indexes.
import "./node_checksum.ts";
import { writeSync } from "node:fs";
import { installRuntimeHost, portableHost } from "./host.ts";
const n = Number(process.env.GOALCHEMY_SEED ?? "1");
installRuntimeHost({
  ...portableHost,
  seed: Number.isInteger(n) && n > 0 && n < 2 ** 32 ? n : 1,
  stdout: bytes => { writeSync(1, bytes); },
  stderr: bytes => { writeSync(2, bytes); },
  fail: code => process.exit(code),
});
