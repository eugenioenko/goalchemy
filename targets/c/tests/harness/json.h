/* Minimal JSON values for the C harness; numbers keep their source text. */
#ifndef GX_JSON_H
#define GX_JSON_H

#include <stdbool.h>
#include <stddef.h>

enum { J_NULL, J_BOOL, J_NUM, J_STR, J_ARR, J_OBJ };

typedef struct J {
    int t;
    bool b;
    char *s;
    size_t n;
    struct J **items;
    char **keys;
    size_t len, cap;
} J;

J *j_parse(const char *s, const char **err);
void j_write(const J *v, char **out, size_t *n, size_t *cap);
J *j_get(const J *o, const char *k);
const char *j_str(const J *v);
J *j_new(int t);
J *j_strn(const char *s, size_t n);
J *j_cstr(const char *s);
J *j_bool(bool b);
void j_push(J *a, J *v);
void j_set(J *o, const char *k, J *v);

#endif
