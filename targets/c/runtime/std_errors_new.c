/* std.errors.new: a distinct *errors.errorString per call. */
#include "gx.h"

static gx_V error_string_text(gx_V *e, gx_V *a, int n) {
    (void)e;
    (void)n;
    return gx_vals(a[0])[0];
}

static bool eq_identity(gx_V a, gx_V b) { return a.u.p == b.u.p; }

static void key_identity(gx_V v, gx_Buf *out) { gx_vkey(v, out); }

static const gx_Method error_string_methods[] = {{"Error", -1, error_string_text}};

static const gx_TypeDesc ERRORS_ERROR_STRING = {0x7fff0010, "*errors.errorString", "pointer", eq_identity, key_identity, error_string_methods, 1, "", true};

gx_V gx_std_errors_new(gx_V text) { return gx_boxv(&ERRORS_ERROR_STRING, gx_new_vals(1, &text)); }
