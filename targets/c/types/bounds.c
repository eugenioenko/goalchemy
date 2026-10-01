/* Go slice-bounds checks and messages. When u is set the bounds have an
 * unsigned 64-bit type: negative values are huge. */
#include "gx.h"

#include <stdio.h>

static void st(char *b, size_t n, int64_t x, bool u) {
    if (u)
        snprintf(b, n, "%llu", (unsigned long long)(uint64_t)x);
    else
        snprintf(b, n, "%lld", (long long)x);
}

static bool neg(int64_t x, bool u) { return !u && x < 0; }

static bool gt(int64_t x, int64_t y, bool u) { return u ? (uint64_t)x > (uint64_t)y : x > y; }

_Noreturn static void oob(const char *fmt, int64_t a, int64_t b, const char *word, int64_t limit, bool u, int form) {
    char sa[24], sb[24], msg[160];
    st(sa, sizeof sa, a, u);
    st(sb, sizeof sb, b, u);
    switch (form) {
    case 0:
        snprintf(msg, sizeof msg, fmt, sa);
        break;
    case 1:
        snprintf(msg, sizeof msg, fmt, sa, word, (long long)limit);
        break;
    default:
        snprintf(msg, sizeof msg, fmt, sa, sb);
    }
    char full[200];
    snprintf(full, sizeof full, "slice bounds out of range %s", msg);
    gx_runtime_panic(full);
}

void gx_check2(int64_t lo, int64_t hi, int64_t limit, const char *word, bool u) {
    if (neg(hi, u)) oob("[:%s]", hi, 0, word, limit, u, 0);
    if (gt(hi, limit, u)) oob("[:%s] with %s %lld", hi, 0, word, limit, u, 1);
    if (neg(lo, u)) oob("[%s:]", lo, 0, word, limit, u, 0);
    if (gt(lo, hi, u)) oob("[%s:%s]", lo, hi, word, limit, u, 2);
}

void gx_check3(int64_t lo, int64_t hi, int64_t max, int64_t limit, const char *word, bool u) {
    if (neg(max, u)) oob("[::%s]", max, 0, word, limit, u, 0);
    if (gt(max, limit, u)) oob("[::%s] with %s %lld", max, 0, word, limit, u, 1);
    if (neg(hi, u)) oob("[:%s:]", hi, 0, word, limit, u, 0);
    if (gt(hi, max, u)) oob("[:%s:%s]", hi, max, word, limit, u, 2);
    if (neg(lo, u)) oob("[%s::]", lo, 0, word, limit, u, 0);
    if (gt(lo, hi, u)) oob("[%s:%s:]", lo, hi, word, limit, u, 2);
}
