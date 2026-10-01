/* UTF-8 over byte strings. */
#include "gx.h"

#define RUNE_ERROR 0xFFFD

int64_t gx_utf8_decode(const uint8_t *s, size_t len, size_t i, size_t *width) {
    size_t n = len - i;
    int64_t b0 = s[i];
    *width = 1;
    if (b0 < 0x80) return b0;
    if (b0 >= 0xC2 && b0 <= 0xDF) {
        if (n < 2) return RUNE_ERROR;
        int64_t b1 = s[i + 1];
        if (b1 < 0x80 || b1 > 0xBF) return RUNE_ERROR;
        *width = 2;
        return ((b0 & 0x1F) << 6) | (b1 & 0x3F);
    }
    if (b0 >= 0xE0 && b0 <= 0xEF) {
        if (n < 3) return RUNE_ERROR;
        int64_t b1 = s[i + 1];
        int64_t lo = b0 == 0xE0 ? 0xA0 : 0x80, hi = b0 == 0xED ? 0x9F : 0xBF;
        if (b1 < lo || b1 > hi) return RUNE_ERROR;
        int64_t b2 = s[i + 2];
        if (b2 < 0x80 || b2 > 0xBF) return RUNE_ERROR;
        *width = 3;
        return ((b0 & 0x0F) << 12) | ((b1 & 0x3F) << 6) | (b2 & 0x3F);
    }
    if (b0 >= 0xF0 && b0 <= 0xF4) {
        if (n < 4) return RUNE_ERROR;
        int64_t b1 = s[i + 1];
        int64_t lo = b0 == 0xF0 ? 0x90 : 0x80, hi = b0 == 0xF4 ? 0x8F : 0xBF;
        if (b1 < lo || b1 > hi) return RUNE_ERROR;
        int64_t b2 = s[i + 2], b3 = s[i + 3];
        if (b2 < 0x80 || b2 > 0xBF || b3 < 0x80 || b3 > 0xBF) return RUNE_ERROR;
        *width = 4;
        return ((b0 & 0x07) << 18) | ((b1 & 0x3F) << 12) | ((b2 & 0x3F) << 6) | (b3 & 0x3F);
    }
    return RUNE_ERROR;
}

void gx_utf8_encode(int64_t r, gx_Buf *out) {
    if (r < 0 || r > 0x10FFFF || (r >= 0xD800 && r <= 0xDFFF)) r = RUNE_ERROR;
    uint32_t c = (uint32_t)r;
    uint8_t b[4];
    size_t n;
    if (c < 0x80) {
        b[0] = (uint8_t)c;
        n = 1;
    } else if (c < 0x800) {
        b[0] = (uint8_t)(0xC0 | (c >> 6));
        b[1] = (uint8_t)(0x80 | (c & 0x3F));
        n = 2;
    } else if (c < 0x10000) {
        b[0] = (uint8_t)(0xE0 | (c >> 12));
        b[1] = (uint8_t)(0x80 | ((c >> 6) & 0x3F));
        b[2] = (uint8_t)(0x80 | (c & 0x3F));
        n = 3;
    } else {
        b[0] = (uint8_t)(0xF0 | (c >> 18));
        b[1] = (uint8_t)(0x80 | ((c >> 12) & 0x3F));
        b[2] = (uint8_t)(0x80 | ((c >> 6) & 0x3F));
        b[3] = (uint8_t)(0x80 | (c & 0x3F));
        n = 4;
    }
    gx_buf_put(out, b, n);
}
