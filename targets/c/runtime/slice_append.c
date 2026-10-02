/* core.slice.append: append with Goalchemy's growth rule; aggregate
 * elements are cloned when they move. */
#include "gx.h"

int64_t gx_grow_cap(int64_t old, int64_t required) {
    int64_t doubled = old <= INT64_MAX / 2 ? 2 * old : INT64_MAX;
    if (doubled < 1) doubled = 1;
    return required > doubled ? required : doubled;
}

/* Returns a value header; selected writes never resize aliased payloads. */
static gx_V byte_room(gx_V s, uint32_t k) {
    uint64_t n = (uint64_t)s.l + k;
    if (n > UINT32_MAX / 2) gx_fault("slice growth exceeds host limits");
    if (n <= s.c) return gx_byte_slice(gx_bytes(s), (uint32_t)n, s.c);
    int64_t c = gx_grow_cap(s.c, (int64_t)n);
    if (c > UINT32_MAX / 2) gx_fault("slice growth exceeds host limits");
    uint8_t *a = gx_alloc_bytes((size_t)c);
    if (s.l) memcpy(a, s.u.p, s.l);
    return gx_byte_slice(a, (uint32_t)n, (uint32_t)c);
}

gx_V gx_append_bytes(gx_V s, const uint8_t *p, uint32_t k) {
    if (!k) return s;
    gx_V r = byte_room(s, k);
    memmove(gx_bytes(r) + s.l, p, k);
    return r;
}

static gx_V append_values(gx_V s, int k, const gx_V *vs, gx_CloneFn clone) {
    if (k < 0) gx_fault("negative append count");
    if (k == 0) return s;
    if (gx_byte_backing(s)) {
        gx_V r = byte_room(s, (uint32_t)k);
        for (int i = 0; i < k; i++) gx_bytes(r)[s.l + (uint32_t)i] = (uint8_t)vs[i].u.i;
        return r;
    }
    size_t n = (size_t)s.l + (size_t)k;
    if (n <= s.c) {
        memcpy(gx_vals(s) + s.l, vs, (size_t)k * sizeof(gx_V));
        return gx_slice(gx_vals(s), (uint32_t)n, s.c);
    }
    int64_t c = gx_grow_cap(s.c, (int64_t)n);
    if (c > (int64_t)(UINT32_MAX / 2)) gx_fault("slice growth exceeds host limits");
    gx_V *a = gx_alloc_vals((size_t)c);
    for (uint32_t i = 0; i < s.l; i++) a[i] = clone ? clone(gx_vals(s)[i]) : gx_vals(s)[i];
    memcpy(a + s.l, vs, (size_t)k * sizeof(gx_V));
    return gx_slice(a, (uint32_t)n, (uint32_t)c);
}

gx_V gx_append(gx_V s, int n, const gx_V *vs, gx_CloneFn clone) { return append_values(s, n, vs, clone); }

gx_V gx_append_slice(gx_V s, gx_V t, gx_CloneFn clone) {
    if (t.l == 0) return s;
    if (gx_byte_backing(s)) {
        if (!gx_byte_backing(t)) gx_fault("byte append requires native source");
        return gx_append_bytes(s, gx_bytes(t), t.l);
    }
    gx_V *vs = gx_alloc_vals(t.l);
    for (uint32_t i = 0; i < t.l; i++) vs[i] = clone ? clone(gx_vals(t)[i]) : gx_vals(t)[i];
    return append_values(s, (int)t.l, vs, clone);
}

gx_V gx_append_string(gx_V s, gx_V str) {
    if (gx_byte_backing(s)) return gx_append_bytes(s, gx_sbytes(str), str.l);
    gx_V *vs = gx_alloc_vals(str.l);
    for (uint32_t i = 0; i < str.l; i++) vs[i] = gx_int(gx_sbytes(str)[i]);
    return append_values(s, (int)str.l, vs, NULL);
}
