// Host file system for lib/os. Node entries install an adapter; portable and
// browser entries leave it unset, so every call reports Unsupported.
export const MAX_FILE_BYTES = 1 << 30;
export const FileStatus = { OK: 0, NotExist: 1, Exist: 2, Permission: 3, IsDir: 4, NotDir: 5, TooLarge: 6, Unsupported: 7, Invalid: 8, IO: 9 } as const;
export type FileStatus = (typeof FileStatus)[keyof typeof FileStatus];
export interface FileHost {
  read(path: Uint8Array): Uint8Array | FileStatus;
  write(path: Uint8Array, data: Uint8Array, mode: number): FileStatus;
}
let fileHost: FileHost | undefined;
export function installFileHost(host: FileHost): void { fileHost = host; }
export function fileStatus(e: unknown): FileStatus {
  switch ((e as { code?: unknown } | null)?.code) {
    case "ENOENT": return FileStatus.NotExist;
    case "EEXIST": return FileStatus.Exist;
    case "EACCES": case "EPERM": return FileStatus.Permission;
    case "EISDIR": return FileStatus.IsDir;
    case "ENOTDIR": return FileStatus.NotDir;
    case "EFBIG": case "ERR_FS_FILE_TOO_LARGE": return FileStatus.TooLarge;
    case "ENOSYS": case "ENOTSUP": case "EOPNOTSUPP": return FileStatus.Unsupported;
    case "EINVAL": case "ERR_INVALID_ARG_VALUE": case "ERR_INVALID_ARG_TYPE": return FileStatus.Invalid;
  }
  return FileStatus.IO;
}
function path(name: string): Uint8Array | FileStatus {
  if (name.length === 0) return FileStatus.NotExist;
  const b = new Uint8Array(name.length);
  for (let i = 0; i < name.length; i++) {
    const c = name.charCodeAt(i);
    if (c === 0 || c > 255) return FileStatus.Invalid;
    b[i] = c;
  }
  return b;
}
export function readHostFile(name: string): Uint8Array | FileStatus {
  const p = path(name);
  if (typeof p === "number") return p;
  if (fileHost === undefined) return FileStatus.Unsupported;
  try { return fileHost.read(p); } catch (e) { return fileStatus(e); }
}
export function writeHostFile(name: string, data: Uint8Array, mode: number): FileStatus {
  const p = path(name);
  if (typeof p === "number") return p;
  if (data.length > MAX_FILE_BYTES) return FileStatus.TooLarge;
  if (fileHost === undefined) return FileStatus.Unsupported;
  try { return fileHost.write(p, data, mode & 0o777); } catch (e) { return fileStatus(e); }
}
