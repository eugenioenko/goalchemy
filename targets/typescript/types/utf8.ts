// UTF-8 over binary strings (one char code per byte).

export const RUNE_ERROR = 0xfffd;

/** Decodes the rune at byte offset i, returning [rune, width]. */
export function decode(s: string, i: number): [number, number] {
  const n = s.length - i;
  const b0 = s.charCodeAt(i);
  if (b0 < 0x80) return [b0, 1];
  const cont = (k: number) => {
    const b = s.charCodeAt(i + k);
    return b >= 0x80 && b <= 0xbf ? b : -1;
  };
  if (b0 >= 0xc2 && b0 <= 0xdf) {
    if (n < 2) return [RUNE_ERROR, 1];
    const b1 = cont(1);
    if (b1 < 0) return [RUNE_ERROR, 1];
    return [((b0 & 0x1f) << 6) | (b1 & 0x3f), 2];
  }
  if (b0 >= 0xe0 && b0 <= 0xef) {
    if (n < 3) return [RUNE_ERROR, 1];
    const b1 = s.charCodeAt(i + 1);
    const lo = b0 === 0xe0 ? 0xa0 : 0x80;
    const hi = b0 === 0xed ? 0x9f : 0xbf;
    if (b1 < lo || b1 > hi) return [RUNE_ERROR, 1];
    const b2 = cont(2);
    if (b2 < 0) return [RUNE_ERROR, 1];
    return [((b0 & 0x0f) << 12) | ((b1 & 0x3f) << 6) | (b2 & 0x3f), 3];
  }
  if (b0 >= 0xf0 && b0 <= 0xf4) {
    if (n < 4) return [RUNE_ERROR, 1];
    const b1 = s.charCodeAt(i + 1);
    const lo = b0 === 0xf0 ? 0x90 : 0x80;
    const hi = b0 === 0xf4 ? 0x8f : 0xbf;
    if (b1 < lo || b1 > hi) return [RUNE_ERROR, 1];
    const b2 = cont(2);
    const b3 = cont(3);
    if (b2 < 0 || b3 < 0) return [RUNE_ERROR, 1];
    return [((b0 & 0x07) << 18) | ((b1 & 0x3f) << 12) | ((b2 & 0x3f) << 6) | (b3 & 0x3f), 4];
  }
  return [RUNE_ERROR, 1];
}

/** Encodes a code point; invalid code points encode U+FFFD. */
export function encode(r: number): string {
  if (r < 0 || r > 0x10ffff || (r >= 0xd800 && r <= 0xdfff)) r = RUNE_ERROR;
  if (r < 0x80) return String.fromCharCode(r);
  if (r < 0x800) return String.fromCharCode(0xc0 | (r >> 6), 0x80 | (r & 0x3f));
  if (r < 0x10000) return String.fromCharCode(0xe0 | (r >> 12), 0x80 | ((r >> 6) & 0x3f), 0x80 | (r & 0x3f));
  return String.fromCharCode(0xf0 | (r >> 18), 0x80 | ((r >> 12) & 0x3f), 0x80 | ((r >> 6) & 0x3f), 0x80 | (r & 0x3f));
}

/** Converts a JavaScript (UTF-16) string to its UTF-8 binary string. */
export function fromHost(s: string): string {
  return Array.from(new TextEncoder().encode(s), b => String.fromCharCode(b)).join("");
}
