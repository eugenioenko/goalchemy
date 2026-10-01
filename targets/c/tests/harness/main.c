/* Serves runtime conformance requests for the C target over JSON Lines on
 * standard input and output. */
#include <stdio.h>
#include <stdlib.h>

#include "codec.h"

static J *serve(J *req) {
    J *resp = j_new(J_OBJ);
    J *v = j_new(J_NUM);
    v->s = "1";
    v->n = 1;
    j_set(resp, "v", v);
    J *id = j_get(req, "id");
    j_set(resp, "id", id ? id : j_new(J_NULL));
    if (strcmp(j_str(j_get(req, "v")), "1") != 0) {
        j_set(resp, "status", j_cstr("harness_failure"));
        j_set(resp, "error", j_cstr("unsupported protocol version"));
        return resp;
    }
    const char *name = j_str(j_get(req, "case"));
    CaseFn fn = NULL;
    for (size_t i = 0; i < NCASES; i++)
        if (strcmp(CASES[i].name, name) == 0) fn = CASES[i].fn;
    if (!fn) {
        char buf[256];
        snprintf(buf, sizeof buf, "unknown case %s", name);
        j_set(resp, "status", j_cstr("harness_failure"));
        j_set(resp, "error", j_cstr(buf));
        return resp;
    }
    H h = {j_get(req, "let"), j_new(J_OBJ)};
    if (!h.lets) h.lets = j_new(J_OBJ);
    gx_reset_scheduler();
    gx_Handler hd;
    if (GX_TRY(hd)) {
        if (gx_blocked_signal) {
            j_set(resp, "status", j_cstr("blocked"));
        } else {
            gx_Buf b = {0};
            gx_format_panic_value(gx_thrown->value, &b);
            j_set(resp, "status", j_cstr("panic"));
            j_set(resp, "panic", j_strn((const char *)b.b, b.n));
        }
        return resp;
    }
    J *results = fn(&h);
    GX_END(hd);
    j_set(resp, "status", j_cstr("returned"));
    j_set(resp, "results", results);
    j_set(resp, "after", h.after);
    return resp;
}

static void loop(void) {
    char *line = NULL;
    size_t cap = 0;
    ssize_t n;
    while ((n = getline(&line, &cap, stdin)) > 0) {
        if (n == 1 && line[0] == '\n') continue;
        const char *err;
        J *req = j_parse(line, &err);
        J *resp;
        if (!req) {
            resp = j_new(J_OBJ);
            j_set(resp, "status", j_cstr("harness_failure"));
            j_set(resp, "error", j_cstr(err));
        } else {
            resp = serve(req);
        }
        char *out = NULL;
        size_t on = 0, oc = 0;
        j_write(resp, &out, &on, &oc);
        fwrite(out, 1, on, stdout);
        fputc('\n', stdout);
        fflush(stdout);
    }
}

int main(void) {
    GC_INIT();
    gx_run_large(loop);
    return 0;
}
