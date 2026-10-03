// The one standard Node API used by generated libraries. This declaration also
// merges with @types/node when a consuming project installs the full package.
declare module "node:zlib" {
  export function crc32(data: Uint8Array, value?: number): number;
}
