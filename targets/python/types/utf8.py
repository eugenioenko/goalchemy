"""UTF-8 decoding and encoding with Go's invalid-sequence behavior."""

RUNE_ERROR = 0xFFFD


def decode(s, i):
    """Decodes the rune at byte offset i of bytes s: (rune, width)."""
    n = len(s) - i
    b0 = s[i]
    if b0 < 0x80:
        return b0, 1

    def cont(k):
        b = s[i + k]
        return b if 0x80 <= b <= 0xBF else -1

    if 0xC2 <= b0 <= 0xDF:
        if n < 2:
            return RUNE_ERROR, 1
        b1 = cont(1)
        if b1 < 0:
            return RUNE_ERROR, 1
        return ((b0 & 0x1F) << 6) | (b1 & 0x3F), 2
    if 0xE0 <= b0 <= 0xEF:
        if n < 3:
            return RUNE_ERROR, 1
        b1 = s[i + 1]
        lo = 0xA0 if b0 == 0xE0 else 0x80
        hi = 0x9F if b0 == 0xED else 0xBF
        if b1 < lo or b1 > hi:
            return RUNE_ERROR, 1
        b2 = cont(2)
        if b2 < 0:
            return RUNE_ERROR, 1
        return ((b0 & 0x0F) << 12) | ((b1 & 0x3F) << 6) | (b2 & 0x3F), 3
    if 0xF0 <= b0 <= 0xF4:
        if n < 4:
            return RUNE_ERROR, 1
        b1 = s[i + 1]
        lo = 0x90 if b0 == 0xF0 else 0x80
        hi = 0x8F if b0 == 0xF4 else 0xBF
        if b1 < lo or b1 > hi:
            return RUNE_ERROR, 1
        b2 = cont(2)
        b3 = cont(3)
        if b2 < 0 or b3 < 0:
            return RUNE_ERROR, 1
        return ((b0 & 0x07) << 18) | ((b1 & 0x3F) << 12) | ((b2 & 0x3F) << 6) | (b3 & 0x3F), 4
    return RUNE_ERROR, 1


def encode(r):
    """Encodes a code point; invalid code points encode U+FFFD."""
    if r < 0 or r > 0x10FFFF or 0xD800 <= r <= 0xDFFF:
        r = RUNE_ERROR
    if r < 0x80:
        return bytes((r,))
    if r < 0x800:
        return bytes((0xC0 | (r >> 6), 0x80 | (r & 0x3F)))
    if r < 0x10000:
        return bytes((0xE0 | (r >> 12), 0x80 | ((r >> 6) & 0x3F), 0x80 | (r & 0x3F)))
    return bytes((0xF0 | (r >> 18), 0x80 | ((r >> 12) & 0x3F), 0x80 | ((r >> 6) & 0x3F), 0x80 | (r & 0x3F)))
