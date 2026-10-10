// Node file-system adapter for lib/os, excluded from every portable runtime index.
import { closeSync, fstatSync, openSync, readFileSync, writeFileSync } from "node:fs";
import { FileStatus, MAX_FILE_BYTES, installFileHost } from "./files.ts";

installFileHost({
  read: path => {
    const fd = openSync(path, "r");
    try {
      if (fstatSync(fd).size > MAX_FILE_BYTES) return FileStatus.TooLarge;
      const data = readFileSync(fd);
      return data.length > MAX_FILE_BYTES ? FileStatus.TooLarge : new Uint8Array(data.buffer, data.byteOffset, data.length);
    } finally {
      closeSync(fd);
    }
  },
  write: (path, data, mode) => {
    writeFileSync(path, data, { mode });
    return FileStatus.OK;
  },
});
