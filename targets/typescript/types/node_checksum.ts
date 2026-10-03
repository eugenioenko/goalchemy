// Node standard-library adapter, excluded from every portable runtime index.
import { crc32 } from "node:zlib";
import { installCRC32IEEE } from "./host.ts";

installCRC32IEEE((bytes: Uint8Array): number => crc32(bytes) >>> 0);
