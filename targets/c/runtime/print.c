/* core.print: print and println of Booleans, integers, and strings to
 * standard error. Unsigned 64-bit operands are formatted by the caller. */
#include "gx.h"

#include <stdio.h>

static void put(gx_V v, gx_Buf *b) {
    char buf[24];
    switch (v.t) {
    case GX_STR:
        gx_buf_put(b, v.u.p, v.l);
        return;
    case GX_BOOL:
        gx_buf_put(b, v.u.i ? "true" : "false", v.u.i ? 4 : 5);
        return;
    case GX_INT: {
        int n = snprintf(buf, sizeof buf, "%lld", (long long)v.u.i);
        gx_buf_put(b, buf, (size_t)n);
        return;
    }
    case GX_NIL:
        gx_buf_put(b, "nil", 3);
        return;
    }
    gx_buf_put(b, "?", 1);
}

gx_V gx_print_string(gx_V v, bool unsigned_) {
    if (unsigned_) return gx_u64s(v);
    gx_Buf b = {0};
    put(v, &b);
    return gx_str((const char *)b.b, b.n);
}

void gx_print(int n, const gx_V *args, bool newline) {
    gx_Buf b = {0};
    for (int i = 0; i < n; i++) {
        if (newline && i > 0) gx_buf_put(&b, " ", 1);
        put(args[i], &b);
    }
    if (newline) gx_buf_put(&b, "\n", 1);
    gx_stderr(b.b, b.n);
}
