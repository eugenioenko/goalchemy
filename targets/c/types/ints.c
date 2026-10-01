/* Shift counts, division by zero, and unsigned helpers. */
#include "gx.h"

#include <stdio.h>

unsigned gx_count(gx_V n) {
    int64_t x = n.u.i;
    if (x < 0) gx_runtime_panic("negative shift amount");
    return x > 64 ? 64 : (unsigned)x;
}

unsigned gx_countu(gx_V n) {
    int64_t x = n.u.i;
    return x < 0 || x > 64 ? 64 : (unsigned)x;
}

_Noreturn void gx_div_zero(void) { gx_runtime_panic("integer divide by zero"); }

gx_V gx_u64s(gx_V v) {
    char buf[24];
    int n = snprintf(buf, sizeof buf, "%llu", (unsigned long long)(uint64_t)v.u.i);
    return gx_str(buf, (size_t)n);
}

int gx_cmpu(gx_V a, gx_V b) {
    uint64_t x = (uint64_t)a.u.i, y = (uint64_t)b.u.i;
    return x < y ? -1 : x > y;
}

int gx_scmp(gx_V a, gx_V b) {
    size_t n = a.l < b.l ? a.l : b.l;
    int c = n ? memcmp(a.u.p, b.u.p, n) : 0;
    if (c) return c < 0 ? -1 : 1;
    return a.l < b.l ? -1 : a.l > b.l;
}
