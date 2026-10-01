/* Function values, dynamic types, and interface boxes. */
#include "gx.h"

#include <stdio.h>

gx_V gx_func(int64_t fid, gx_Code code, int n, const gx_V *env) {
    gx_Func *f = GC_MALLOC(sizeof(gx_Func) + (size_t)n * sizeof(gx_V));
    f->fid = fid;
    f->code = code;
    f->n = n;
    if (n) memcpy(f->env, env, (size_t)n * sizeof(gx_V));
    return gx_obj(f);
}

gx_V gx_callv(gx_V f, int n, const gx_V *args) {
    if (f.t != GX_OBJ) gx_panic_nil_deref();
    gx_Func *fn = f.u.p;
    gx_V *a = gx_alloc_vals((size_t)n);
    if (n) memcpy(a, args, (size_t)n * sizeof(gx_V));
    return fn->code(fn->env, a, n);
}

gx_V gx_fnchk(gx_V f) {
    if (f.t != GX_OBJ) gx_panic_nil_deref();
    return f;
}

int64_t gx_fid_of(gx_V f) { return f.t == GX_OBJ ? ((gx_Func *)f.u.p)->fid : -1; }

static gx_V bound_code(gx_V *env, gx_V *args, int n) {
    gx_V *all = gx_alloc_vals((size_t)n + 1);
    all[0] = env[1];
    if (n) memcpy(all + 1, args, (size_t)n * sizeof(gx_V));
    return gx_callv(env[0], n + 1, all);
}

gx_V gx_bound(int64_t fid, gx_V f, gx_V recv) {
    gx_V env[2] = {f, recv};
    return gx_func(fid, bound_code, 2, env);
}

const gx_Method *gx_method(const gx_TypeDesc *t, const char *id) {
    for (int i = 0; i < t->nmethods; i++)
        if (strcmp(t->methods[i].id, id) == 0) return &t->methods[i];
    return NULL;
}

gx_V gx_boxv(const gx_TypeDesc *t, gx_V v) {
    gx_Box *b = GC_MALLOC(sizeof(gx_Box));
    b->t = t;
    b->v = v;
    return gx_obj(b);
}

const gx_TypeDesc *gx_dyn_type(gx_V x) { return x.t == GX_OBJ ? ((gx_Box *)x.u.p)->t : NULL; }

gx_V gx_unboxed(gx_V x) { return x.t == GX_OBJ ? ((gx_Box *)x.u.p)->v : gx_nil(); }

bool gx_ifeq(gx_V a, gx_V b) {
    const gx_TypeDesc *ta = gx_dyn_type(a), *tb = gx_dyn_type(b);
    if (!ta || !tb) return ta == tb;
    return ta == tb && ta->eq(gx_unboxed(a), gx_unboxed(b));
}

void gx_ikey(gx_V a, gx_Buf *out) {
    const gx_TypeDesc *t = gx_dyn_type(a);
    if (!t) {
        gx_buf_put(out, "N", 1);
        return;
    }
    gx_buf_put(out, "X", 1);
    gx_buf_put(out, &t->id, 4);
    t->key(gx_unboxed(a), out);
}

const char *gx_implements_all(const gx_TypeDesc *t, int n, const char *const *ids) {
    for (int i = 0; i < n; i++)
        if (!gx_method(t, ids[i])) return ids[i];
    return NULL;
}

bool gx_implements(gx_V x, int n, const char *const *ids) {
    const gx_TypeDesc *t = gx_dyn_type(x);
    return t && gx_implements_all(t, n, ids) == NULL;
}

const char *gx_missing_method(gx_V x, int n, const char *const *ids, const char *const *names) {
    const gx_TypeDesc *t = gx_dyn_type(x);
    if (!t) return NULL;
    const char *id = gx_implements_all(t, n, ids);
    for (int i = 0; id && i < n; i++)
        if (ids[i] == id) return names[i];
    return NULL;
}

bool gx_is_type(gx_V x, const gx_TypeDesc *t) { return gx_dyn_type(x) == t; }

gx_V gx_icall(gx_V x, const char *id, int n, const gx_V *args) {
    if (x.t != GX_OBJ) gx_panic_nil_deref();
    gx_Box *b = x.u.p;
    const gx_Method *m = gx_method(b->t, id);
    if (!m) gx_fault("missing method");
    gx_V *all = gx_alloc_vals((size_t)n + 1);
    all[0] = b->v;
    if (n) memcpy(all + 1, args, (size_t)n * sizeof(gx_V));
    return m->code(NULL, all, n + 1);
}

gx_V gx_ibound(gx_V x, const char *id) {
    if (x.t != GX_OBJ) gx_panic_nil_deref();
    gx_Box *b = x.u.p;
    const gx_Method *m = gx_method(b->t, id);
    if (!m) gx_fault("missing method");
    return gx_bound(m->fid, gx_func(m->fid, m->code, 0, NULL), b->v);
}

bool gx_eq_uncomparable(const char *name) {
    char buf[256];
    snprintf(buf, sizeof buf, "comparing uncomparable type %s", name);
    gx_runtime_panic(buf);
}

void gx_key_unhashable(const char *name) {
    char buf[256];
    snprintf(buf, sizeof buf, "hash of unhashable type %s", name);
    gx_runtime_panic(buf);
}

bool gx_eq_basic(gx_V a, gx_V b) { return gx_veq(a, b); }
