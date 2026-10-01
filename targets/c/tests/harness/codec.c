#include "codec.h"

#include <stdio.h>
#include <stdlib.h>

J *h_let(H *h, const char *n) {
    J *v = j_get(h->lets, n);
    return v ? v : j_new(J_NULL);
}

void h_after(H *h, const char *n, J *v) { j_set(h->after, n, v); }

static bool is_nil(J *r) {
    J *n = j_get(r, "nil");
    return n && n->t == J_BOOL && n->b;
}

static long long num(J *r) { return r ? strtoll(j_str(r), NULL, 10) : 0; }

gx_V dec_int(J *r) {
    const char *s = j_str(r);
    if (*s == '-') return gx_int(strtoll(s, NULL, 10));
    return gx_int((int64_t)strtoull(s, NULL, 10));
}

gx_V dec_bool(J *r) { return gx_bool((r->t == J_BOOL && r->b) || strcmp(j_str(r), "true") == 0); }

gx_V dec_string(J *r) {
    J *s = j_get(r, "str");
    if (s) return gx_str(s->s, s->n);
    const char *h = j_str(j_get(r, "hex"));
    size_t n = strlen(h) / 2;
    char *b = GC_MALLOC_ATOMIC(n ? n : 1);
    for (size_t i = 0; i < n; i++) {
        char t[3] = {h[2 * i], h[2 * i + 1], 0};
        b[i] = (char)strtoul(t, NULL, 16);
    }
    return gx_str(b, n);
}

gx_V dec_error(J *r) {
    if (is_nil(r)) return gx_nil();
    J *e = j_get(r, "error");
    return gx_std_errors_new(gx_str(e->s, e->n));
}

gx_V dec_nil(J *r) { return gx_nil(); }

gx_V dec_slice(J *r, DecFn dec, gx_ZeroFn zero) {
    if (is_nil(r)) return gx_nil_slice();
    J *items = j_get(r, "slice");
    size_t n = items->len;
    J *c = j_get(r, "cap");
    size_t cap = c ? (size_t)num(c) : n;
    gx_V *a = gx_alloc_vals(cap ? cap : 1);
    for (size_t i = 0; i < cap; i++) a[i] = i < n ? dec(items->items[i]) : zero();
    return gx_slice(a, (uint32_t)n, (uint32_t)cap);
}

gx_V view(gx_V base, J *r) {
    J *lo = j_get(r, "lo"), *hi = j_get(r, "hi"), *mx = j_get(r, "max");
    uint32_t l = lo ? (uint32_t)num(lo) : 0;
    uint32_t h = hi ? (uint32_t)num(hi) : base.l;
    uint32_t m = mx ? (uint32_t)num(mx) : base.c;
    return gx_slice(gx_vals(base) + l, h - l, m - l);
}

gx_V dec_map(J *r, DecFn dk, DecFn dv) {
    if (is_nil(r)) return gx_nil();
    gx_V m = gx_make_map(gx_vkey);
    J *es = j_get(r, "map");
    for (size_t i = 0; i < es->len; i++) gx_map_set(m, dk(j_get(es->items[i], "key")), dv(j_get(es->items[i], "value")));
    return m;
}

gx_V dec_chan(J *r, DecFn dec, gx_ZeroFn zero) {
    if (is_nil(r)) return gx_nil();
    J *c = j_get(r, "cap");
    gx_V ch = gx_make_chan(gx_int(c ? num(c) : 0), zero);
    gx_Chan *x = ch.u.p;
    J *items = j_get(r, "chan");
    size_t n = items ? items->len : 0;
    if (n > x->size) {
        gx_V *b = gx_alloc_vals(n);
        x->buf = b;
    }
    for (size_t i = 0; i < n; i++) x->buf[i] = dec(items->items[i]);
    x->len = n;
    if (n > x->size) x->size = n;
    J *cl = j_get(r, "closed");
    x->closed = cl && cl->t == J_BOOL && cl->b;
    return ch;
}

J *enc_int_s(gx_V v) {
    char b[32];
    int n = snprintf(b, sizeof b, "%lld", (long long)v.u.i);
    return j_strn(b, (size_t)n);
}

J *enc_int_u(gx_V v) {
    char b[32];
    int n = snprintf(b, sizeof b, "%llu", (unsigned long long)(uint64_t)v.u.i);
    return j_strn(b, (size_t)n);
}

J *enc_bool(gx_V v) { return j_cstr(gx_b(v) ? "true" : "false"); }

J *enc_string(gx_V v) {
    char *h = GC_MALLOC_ATOMIC(2 * (size_t)v.l + 1);
    for (uint32_t i = 0; i < v.l; i++) snprintf(h + 2 * i, 3, "%02x", gx_sbytes(v)[i]);
    h[2 * v.l] = 0;
    J *o = j_new(J_OBJ);
    j_set(o, "hex", j_strn(h, 2 * (size_t)v.l));
    return o;
}

static J *nil_obj(void) {
    J *o = j_new(J_OBJ);
    j_set(o, "nil", j_bool(true));
    return o;
}

J *enc_error(gx_V v) {
    const gx_TypeDesc *t = gx_dyn_type(v);
    if (!t) return nil_obj();
    gx_V a[1] = {gx_unboxed(v)};
    gx_V s = gx_method(t, "Error")->code(NULL, a, 1);
    J *o = j_new(J_OBJ);
    j_set(o, "error", j_strn((const char *)gx_sbytes(s), s.l));
    return o;
}

J *enc_zero(gx_V v) {
    J *o = j_new(J_OBJ);
    j_set(o, "zero", j_bool(true));
    return o;
}

static J *cap_str(size_t c) {
    char b[32];
    int n = snprintf(b, sizeof b, "%zu", c);
    return j_strn(b, (size_t)n);
}

J *enc_slice(gx_V v, EncFn enc) {
    if (!v.u.p) return nil_obj();
    J *a = j_new(J_ARR);
    for (uint32_t i = 0; i < v.l; i++) j_push(a, enc(gx_vals(v)[i]));
    J *o = j_new(J_OBJ);
    j_set(o, "slice", a);
    j_set(o, "cap", cap_str(v.c));
    return o;
}

J *enc_array(gx_V v, size_t n, EncFn enc) {
    J *a = j_new(J_ARR);
    for (size_t i = 0; i < n; i++) j_push(a, enc(gx_vals(v)[i]));
    J *o = j_new(J_OBJ);
    j_set(o, "array", a);
    return o;
}

J *enc_map(gx_V v, EncFn ek, EncFn ev) {
    if (v.t == GX_NIL) return nil_obj();
    gx_V it = gx_map_iter(v);
    J *a = j_new(J_ARR);
    while (gx_b(gx_map_next(it))) {
        J *e = j_new(J_OBJ);
        j_set(e, "key", ek(gx_iter_key(it)));
        j_set(e, "value", ev(gx_iter_val(it)));
        j_push(a, e);
    }
    J *o = j_new(J_OBJ);
    j_set(o, "map", a);
    return o;
}

J *enc_chan(gx_V v, EncFn enc) {
    if (v.t == GX_NIL) return nil_obj();
    gx_Chan *c = v.u.p;
    J *a = j_new(J_ARR);
    for (size_t i = 0; i < c->len; i++) j_push(a, enc(c->buf[(c->head + i) % c->size]));
    J *o = j_new(J_OBJ);
    j_set(o, "chan", a);
    j_set(o, "cap", cap_str(c->size));
    j_set(o, "closed", j_bool(c->closed));
    return o;
}

static gx_V mark(gx_V *e, gx_V *a, int n) {
    gx_vals(e[0])[0] = gx_bool(true);
    return gx_nil();
}

/* Reports whether a spawned task ran before the parent yielded. */
static void spawn_check_step(gx_Task *t, gx_Frame *f) {
    if (f->pc == 0) {
        gx_V flag = gx_new_vals(1, (gx_V[]){gx_bool(false)});
        f->l[0] = flag;
        gx_spawn(gx_sync_frame(gx_func(-1, mark, 1, &flag), 0, NULL, 0));
        f->l[1] = gx_vals(flag)[0];
        f->pc = 1;
        gx_std_runtime_gosched(t);
        return;
    }
    if (!gx_b(gx_vals(f->l[0])[0])) gx_fault("spawned task did not run after the parent yielded");
    gx_ret(t, f);
}

static int spawn_check_results(gx_Frame *f, gx_V *out) {
    out[0] = f->l[1];
    return 1;
}

gx_V new_spawn_check(void) { return gx_vframe(gx_new_frame(2, spawn_check_step, spawn_check_results)); }

void harness_select2(gx_Task *t, gx_V a, gx_V b, gx_V dflt) {
    gx_select(t, gx_b(dflt), 2, (gx_Case[]){{a, false, gx_nil()}, {b, false, gx_nil()}});
}
