/* std.errors.unwrap. */
#include "gx.h"

gx_V gx_std_errors_unwrap(gx_V err) {
    const gx_TypeDesc *t = gx_dyn_type(err);
    if (!t) return gx_nil();
    const gx_Method *m = gx_method(t, "Unwrap");
    if (!m) return gx_nil();
    gx_V a[1] = {gx_unboxed(err)};
    return m->code(NULL, a, 1);
}
