/* Execute real native backing operations, including collector roots. */
#include "gx.h"
#include <assert.h>
#include <stdio.h>
#include <stdlib.h>

static gx_V v(gx_V x, int64_t lo, int64_t hi, int64_t max) {
    return gx_reslice(x, gx_int(lo), gx_int(hi), gx_int(max), false);
}
static void same(gx_V a, gx_V b) {
    assert(a.t == b.t && a.pad == b.pad && a.u.p == b.u.p && a.l == b.l && a.c == b.c);
}
static void payload(gx_V x, const uint8_t *p, size_t n) {
    assert(gx_byte_backing(x) && sizeof(*gx_bytes(x)) == 1);
    assert(GC_base(x.u.p));
    /* A cast alone could conceal gx_V allocations. Inspect actual collector
     * object size; these checks use base-backed native payloads. */
    assert(GC_size(GC_base(x.u.p)) >= (n ? n : 1));
    assert(GC_size(GC_base(x.u.p)) < (n ? n : 1) + 64);
    assert(!n || !memcmp(x.u.p, p, n));
}
#define CHECK_PANIC(expr) do { \
    gx_Handler h; \
    if (GX_TRY(h)) { assert(gx_dyn_type(gx_thrown->value) == &GX_RUNTIME_ERROR); gx_thrown = NULL; } \
    else { expr; assert(!"expected source runtime panic"); } \
    GX_END(h); \
} while (0)

static void semantics(void) {
    assert(GC_get_all_interior_pointers());
    gx_V nil = gx_zero_byte_slice(), empty = gx_make_byte_slice(gx_int(0), gx_int(0));
    assert(gx_nil_slice_p(nil) && !gx_nil_slice_p(empty));
    assert(gx_byte_backing(nil) && gx_byte_backing(empty));
    gx_V choices[] = {nil, empty};
    for (size_t i = 0; i < 2; i++) {
        gx_V x = choices[i];
        same(x, v(x, 0, 0, 0));
        same(x, gx_append(x, 0, NULL, NULL));
        same(x, gx_append_string(x, gx_str(NULL, 0)));
        for (size_t j = 0; j < 2; j++) {
            same(x, gx_append_slice(x, choices[j], NULL));
            assert(gx_i(gx_copy(x, choices[j], NULL)) == 0);
            assert(gx_i(gx_copy(choices[j], x, NULL)) == 0);
        }
        gx_clear_slice(x, gx_zero_int);
        assert(gx_i(gx_copy_string(x, gx_cstr("abc"))) == 0);
        assert(gx_i(gx_copy_string(x, gx_str(NULL, 0))) == 0);
        assert(gx_from_bytes(x).l == 0);
        gx_V z = gx_slice_to_array(x, 0, NULL), c = gx_byte_array_clone(z, 0);
        assert(z.u.p && c.u.p && z.u.p != c.u.p && gx_byte_backing(z));
        gx_byte_array_set(z, c, 0);
        assert(gx_byte_array_eq(z, c, 0));
    }
    gx_V x = gx_make_byte_slice(gx_int(2), gx_int(6));
    payload(x, (uint8_t[]){0, 0, 0, 0, 0, 0}, 6);
    gx_sset(x, gx_int(0), gx_int(-1)); gx_ssetu(x, gx_int(1), gx_int(384));
    assert(gx_i(gx_sget(x, gx_int(0))) == 255 && gx_i(gx_sgetu(x, gx_int(1))) == 128);
    gx_V selected = v(x, 1, 2, 5);
    gx_V r = gx_append(selected, 2, (gx_V[]){gx_int(3), gx_int(4)}, NULL);
    assert(selected.l == 1 && selected.c == 4 && r.l == 3 && r.u.p == selected.u.p);
    payload(x, (uint8_t[]){255, 128, 3, 4, 0, 0}, 6);
    gx_V grown = gx_append_slice(r, r, NULL);
    assert(grown.u.p != r.u.p && grown.c == 8 && grown.l == 6);
    payload(grown, (uint8_t[]){128, 3, 4, 128, 3, 4, 0, 0}, 8);
    gx_sset(grown, gx_int(0), gx_int(9)); assert(gx_i(gx_sget(r, gx_int(0))) == 128);
    gx_V empties[] = {v(x, 1, 1, 6), v(x, 6, 6, 6)};
    for (size_t i = 0; i < 2; i++) {
        gx_clear_slice(empties[i], gx_zero_int);
        assert(gx_i(gx_copy_string(empties[i], gx_str(NULL, 0))) == 0);
        assert(gx_i(gx_copy(empties[i], x, NULL)) == 0);
        assert(gx_i(gx_copy(x, empties[i], NULL)) == 0);
        assert(gx_i(gx_copy_string(empties[i], gx_cstr("abc"))) == 0);
        same(empties[i], gx_append_string(empties[i], gx_str(NULL, 0)));
    }
    assert(gx_i(gx_copy_string(x, gx_str(NULL, 0))) == 0);
    gx_clear_slice(v(x, 1, 3, 6), gx_zero_int);
    payload(x, (uint8_t[]){255, 0, 0, 4, 0, 0}, 6);
    CHECK_PANIC(gx_ssetu(x, gx_int(INT64_MIN), gx_int(1)));
    payload(x, (uint8_t[]){255, 0, 0, 4, 0, 0}, 6);
    CHECK_PANIC(gx_sgetu(x, gx_int(INT64_MIN)));
    CHECK_PANIC(gx_reslice(x, gx_int(0), gx_int(INT64_MIN), gx_nil(), true));
    CHECK_PANIC(gx_make_byte_slice(gx_int(-1), gx_int(0)));
    CHECK_PANIC(gx_make_byte_slice(gx_int(2), gx_int(1)));
    CHECK_PANIC(gx_make_byte_slice(gx_int(INT64_MIN), gx_int(INT64_MIN)));
    CHECK_PANIC(gx_slice_to_array(x, 3, NULL));
    gx_V a = gx_byte_array(3);
    CHECK_PANIC(gx_asetu(a, gx_int(INT64_MIN), 3, gx_int(99)));
    payload(a, (uint8_t[]){0,0,0}, 3);
    CHECK_PANIC(gx_agetu(a, gx_int(INT64_MIN), 3));
    gx_V generic = gx_make_slice(gx_int(2), gx_int(4), gx_zero_int);
    gx_sset(generic, gx_int(1), gx_int(300));
    assert(!gx_byte_backing(generic) && gx_i(gx_sget(generic, gx_int(1))) == 300);
    puts("native headers, empty paths, zero capacity, bounds and generic storage passed");
}

static void binary_overlap(void) {
    gx_V x = gx_to_bytes(gx_str("\1\2\3\4\5\6", 6));
    assert(gx_i(gx_copy(v(x,1,6,6), x, NULL)) == 5);
    payload(x, (uint8_t[]){1,1,2,3,4,5}, 6);
    assert(gx_i(gx_copy(x, v(x,2,6,6), NULL)) == 4);
    payload(x, (uint8_t[]){2,3,4,5,4,5}, 6);
    for (int d = 0; d < 2; d++) {
        gx_V b = gx_to_bytes(gx_str("\1\2\3\4\5\6", 6));
        gx_V r = gx_append_slice(v(b,d,d+2,6), v(b,1-d,4-d,6), NULL);
        assert(r.u.p == gx_bytes(b)+d);
        payload(b, d ? (uint8_t[]){1,2,3,1,2,3} : (uint8_t[]){1,2,2,3,4,6}, 6);
    }
    const uint8_t raw[] = {0,255,128,192,175,65};
    gx_V str = gx_str((const char *)raw, sizeof raw), b = gx_to_bytes(str), saved = gx_from_bytes(b);
    assert(b.u.p != str.u.p && saved.u.p != b.u.p);
    gx_sset(b,gx_int(0),gx_int(44));
    assert(!memcmp(saved.u.p,raw,6) && !memcmp(str.u.p,raw,6));
    gx_V again = gx_to_bytes(saved); gx_sset(again,gx_int(1),gx_int(1));
    assert(!memcmp(saved.u.p,raw,6));
    gx_V appended = gx_append_string(v(b,0,1,1),saved);
    payload(appended,(uint8_t[]){44,0,255,128,192,175,65},7);
    gx_copy_string(v(appended,2,7,7),saved);
    payload(appended,(uint8_t[]){44,0,0,255,128,192,175},7);
    puts("overlap both ways and independent binary string copies passed");
}

/* Each mode is launched in its own process with its own allocation. None can
 * retain another mode's payload and conceal a missing collector root shape. */
static gx_V capture(gx_V *env, gx_V *args, int n) { return env[0]; }
static const gx_TypeDesc BYTE_DESC = {123, "Bytes", "slice", NULL, NULL, NULL, 0, "", false};
__attribute__((noinline)) static gx_V make_root(int mode) {
    gx_V b = gx_make_byte_slice(gx_int(4096),gx_int(4096));
    for (uint32_t i = 0; i < b.l; i++) gx_bytes(b)[i] = (uint8_t)(i * 17 + mode * 31);
    if (mode == 0) { /* Escaped whole-array pointer. */
        gx_V a = gx_slice_to_array(b,4096,NULL); return a;
    }
    if (mode == 1) return v(b,37,4037,4096); /* Sole live interior view. */
    if (mode == 2) return gx_new_vals(1,&b); /* Field. */
    if (mode == 3) return gx_boxv(&BYTE_DESC,b); /* Interface. */
    if (mode == 4) return gx_func(123,capture,1,&b); /* Closure. */
    if (mode == 5) { gx_V a = gx_slice_to_array(b,4096,NULL); return gx_byte_array_clone(a,4096); }
    if (mode == 6) return b; /* Direct native slice. */
    if (mode == 7) { gx_V a = gx_byte_array(4096); gx_byte_array_set(a,b,4096); return a; }
    gx_Frame *f = gx_new_frame(1,NULL,NULL);
    assert(f);
    f->l[0] = b; gx_V r = gx_obj(f); r.t = GX_FRAME; return r;
}
__attribute__((noinline)) static void collect_churn(void) {
    /* Overwrite the allocating call's obsolete stack slots before collections. */
    volatile uintptr_t scrub[16384];
    for (size_t i = 0; i < sizeof scrub/sizeof scrub[0]; i++) scrub[i] = 0;
    for (int j = 0; j < 8; j++) {
        for (int i = 0; i < 256; i++) { uint8_t *p = gx_alloc_bytes(32768); p[0] = (uint8_t)i; }
        GC_gcollect();
    }
    assert(scrub[0] == 0);
}
static void gc_mode(int mode) {
    assert(GC_get_all_interior_pointers());
    volatile gx_V root = make_root(mode);
    size_t before = GC_get_gc_no();
    collect_churn();
    gx_V r = root, b = r;
    if (mode == 2) b = gx_fld(r,0);
    if (mode == 3) b = gx_unboxed(r);
    if (mode == 4) b = gx_callv(r,0,NULL);
    if (mode == 8) b = ((gx_Frame *)r.u.p)->l[0];
    assert(gx_byte_backing(b) && GC_base(b.u.p));
    assert(GC_size(GC_base(b.u.p)) >= 4096 && GC_size(GC_base(b.u.p)) < 4160);
    size_t offset = mode == 1 ? 37 : 0, n = mode == 1 ? 4000 : 4096;
    for (size_t i = 0; i < n; i++) assert(gx_bytes(b)[i] == (uint8_t)((i+offset)*17 + mode*31));
    assert(GC_get_gc_no() >= before+8);
    printf("collector root %d: interior=%d collections=%zu payload=%zu passed\n",mode,GC_get_all_interior_pointers(),(size_t)GC_get_gc_no()-before,n);
}
int gx_byte_storage_test(int argc, char **argv) {
    GC_INIT();
    if (argc == 3 && !strcmp(argv[1],"gc")) { gc_mode(atoi(argv[2])); return 0; }
    if (argc == 2 && !strcmp(argv[1],"fault-generic-access")) (void)gx_vals(gx_nil_byte_slice());
    if (argc == 2 && !strcmp(argv[1],"fault-make")) gx_make_byte_slice(gx_int(0),gx_int(UINT32_MAX));
    if (argc == 2 && !strcmp(argv[1],"fault-growth")) {
        gx_V s = gx_byte_slice(NULL,UINT32_MAX/2,UINT32_MAX/2);
        gx_append(s,1,(gx_V[]){gx_int(1)},NULL);
    }
    semantics(); binary_overlap(); return 0;
}
#ifdef GX_BYTE_STANDALONE
int main(int argc, char **argv) { return gx_byte_storage_test(argc,argv); }
#endif
