#include "json.h"

#include <gc.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

J *j_new(int t) {
    J *v = GC_MALLOC(sizeof(J));
    v->t = t;
    return v;
}

J *j_strn(const char *s, size_t n) {
    J *v = j_new(J_STR);
    v->s = GC_MALLOC_ATOMIC(n + 1);
    memcpy(v->s, s, n);
    v->s[n] = 0;
    v->n = n;
    return v;
}

J *j_cstr(const char *s) { return j_strn(s, strlen(s)); }

J *j_bool(bool b) {
    J *v = j_new(J_BOOL);
    v->b = b;
    return v;
}

static void grow(J *a) {
    if (a->len < a->cap) return;
    size_t c = a->cap ? a->cap * 2 : 4;
    J **it = GC_MALLOC(c * sizeof(J *));
    char **ks = GC_MALLOC(c * sizeof(char *));
    if (a->len) {
        memcpy(it, a->items, a->len * sizeof(J *));
        if (a->keys) memcpy(ks, a->keys, a->len * sizeof(char *));
    }
    a->items = it;
    a->keys = ks;
    a->cap = c;
}

void j_push(J *a, J *v) {
    grow(a);
    a->items[a->len++] = v;
}

void j_set(J *o, const char *k, J *v) {
    grow(o);
    o->keys[o->len] = (char *)k;
    o->items[o->len++] = v;
}

J *j_get(const J *o, const char *k) {
    if (!o || o->t != J_OBJ) return NULL;
    for (size_t i = 0; i < o->len; i++)
        if (strcmp(o->keys[i], k) == 0) return o->items[i];
    return NULL;
}

const char *j_str(const J *v) { return v && (v->t == J_STR || v->t == J_NUM) ? v->s : ""; }

typedef struct P {
    const char *s;
    const char *err;
} P;

static void ws(P *p) {
    while (*p->s == ' ' || *p->s == '\t' || *p->s == '\n' || *p->s == '\r') p->s++;
}

static void utf8(char **b, size_t *n, size_t *cap, unsigned c) {
    char t[4];
    size_t k;
    if (c < 0x80) {
        t[0] = (char)c;
        k = 1;
    } else if (c < 0x800) {
        t[0] = (char)(0xC0 | (c >> 6));
        t[1] = (char)(0x80 | (c & 0x3F));
        k = 2;
    } else {
        t[0] = (char)(0xE0 | (c >> 12));
        t[1] = (char)(0x80 | ((c >> 6) & 0x3F));
        t[2] = (char)(0x80 | (c & 0x3F));
        k = 3;
    }
    while (*n + k + 1 > *cap) {
        *cap = *cap ? *cap * 2 : 32;
        char *nb = GC_MALLOC_ATOMIC(*cap);
        if (*n) memcpy(nb, *b, *n);
        *b = nb;
    }
    memcpy(*b + *n, t, k);
    *n += k;
}

static J *string(P *p) {
    if (*p->s != '"') {
        p->err = "expected string";
        return NULL;
    }
    p->s++;
    char *b = NULL;
    size_t n = 0, cap = 0;
    while (*p->s && *p->s != '"') {
        unsigned c = (unsigned char)*p->s++;
        if (c == '\\') {
            char e = *p->s++;
            switch (e) {
            case 'n': c = '\n'; break;
            case 't': c = '\t'; break;
            case 'r': c = '\r'; break;
            case 'b': c = '\b'; break;
            case 'f': c = '\f'; break;
            case 'u': {
                char h[5] = {0};
                memcpy(h, p->s, 4);
                p->s += 4;
                utf8(&b, &n, &cap, (unsigned)strtoul(h, NULL, 16));
                continue;
            }
            default: c = (unsigned char)e;
            }
            utf8(&b, &n, &cap, c);
            continue;
        }
        if (n + 2 > cap) {
            cap = cap ? cap * 2 : 32;
            char *nb = GC_MALLOC_ATOMIC(cap);
            if (n) memcpy(nb, b, n);
            b = nb;
        }
        b[n++] = (char)c;
    }
    if (*p->s != '"') {
        p->err = "unterminated string";
        return NULL;
    }
    p->s++;
    return j_strn(b ? b : "", n);
}

static J *value(P *p) {
    ws(p);
    switch (*p->s) {
    case '{': {
        p->s++;
        J *o = j_new(J_OBJ);
        ws(p);
        if (*p->s == '}') {
            p->s++;
            return o;
        }
        for (;;) {
            ws(p);
            J *k = string(p);
            if (!k) return NULL;
            ws(p);
            if (*p->s != ':') {
                p->err = "expected :";
                return NULL;
            }
            p->s++;
            J *v = value(p);
            if (!v) return NULL;
            j_set(o, k->s, v);
            ws(p);
            if (*p->s == ',') {
                p->s++;
                continue;
            }
            if (*p->s != '}') {
                p->err = "expected }";
                return NULL;
            }
            p->s++;
            return o;
        }
    }
    case '[': {
        p->s++;
        J *a = j_new(J_ARR);
        ws(p);
        if (*p->s == ']') {
            p->s++;
            return a;
        }
        for (;;) {
            J *v = value(p);
            if (!v) return NULL;
            j_push(a, v);
            ws(p);
            if (*p->s == ',') {
                p->s++;
                continue;
            }
            if (*p->s != ']') {
                p->err = "expected ]";
                return NULL;
            }
            p->s++;
            return a;
        }
    }
    case '"':
        return string(p);
    case 't':
        p->s += 4;
        return j_bool(true);
    case 'f':
        p->s += 5;
        return j_bool(false);
    case 'n':
        p->s += 4;
        return j_new(J_NULL);
    }
    const char *st = p->s;
    while (*p->s && strchr("+-0123456789.eE", *p->s)) p->s++;
    if (st == p->s) {
        p->err = "unexpected character";
        return NULL;
    }
    J *v = j_strn(st, (size_t)(p->s - st));
    v->t = J_NUM;
    return v;
}

J *j_parse(const char *s, const char **err) {
    P p = {s, NULL};
    J *v = value(&p);
    ws(&p);
    if (v && *p.s) p.err = "trailing data";
    *err = p.err;
    return p.err ? NULL : v;
}

static void put(char **out, size_t *n, size_t *cap, const char *s, size_t k) {
    while (*n + k + 1 > *cap) {
        *cap = *cap ? *cap * 2 : 256;
        char *nb = GC_MALLOC_ATOMIC(*cap);
        if (*n) memcpy(nb, *out, *n);
        *out = nb;
    }
    memcpy(*out + *n, s, k);
    *n += k;
    (*out)[*n] = 0;
}

static void wstr(const char *s, size_t len, char **out, size_t *n, size_t *cap) {
    put(out, n, cap, "\"", 1);
    for (size_t i = 0; i < len; i++) {
        unsigned char c = (unsigned char)s[i];
        char buf[8];
        if (c == '"' || c == '\\') {
            buf[0] = '\\';
            buf[1] = (char)c;
            put(out, n, cap, buf, 2);
        } else if (c < 0x20) {
            int k = snprintf(buf, sizeof buf, "\\u%04x", c);
            put(out, n, cap, buf, (size_t)k);
        } else {
            put(out, n, cap, (const char *)&s[i], 1);
        }
    }
    put(out, n, cap, "\"", 1);
}

void j_write(const J *v, char **out, size_t *n, size_t *cap) {
    switch (v->t) {
    case J_NULL:
        put(out, n, cap, "null", 4);
        return;
    case J_BOOL:
        put(out, n, cap, v->b ? "true" : "false", v->b ? 4 : 5);
        return;
    case J_NUM:
        put(out, n, cap, v->s, v->n);
        return;
    case J_STR:
        wstr(v->s, v->n, out, n, cap);
        return;
    case J_ARR:
        put(out, n, cap, "[", 1);
        for (size_t i = 0; i < v->len; i++) {
            if (i) put(out, n, cap, ",", 1);
            j_write(v->items[i], out, n, cap);
        }
        put(out, n, cap, "]", 1);
        return;
    case J_OBJ:
        put(out, n, cap, "{", 1);
        for (size_t i = 0; i < v->len; i++) {
            if (i) put(out, n, cap, ",", 1);
            wstr(v->keys[i], strlen(v->keys[i]), out, n, cap);
            put(out, n, cap, ":", 1);
            j_write(v->items[i], out, n, cap);
        }
        put(out, n, cap, "}", 1);
        return;
    }
}
