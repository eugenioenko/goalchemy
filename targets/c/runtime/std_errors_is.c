/* std.errors.is: walks the Unwrap chain comparing and calling Is methods. */
#include "gx.h"

gx_V gx_std_errors_is(gx_V err, gx_V target) {
    if (err.t == GX_NIL || target.t == GX_NIL) return gx_bool(err.t == GX_NIL && target.t == GX_NIL);
    const gx_TypeDesc *tt = gx_dyn_type(target);
    gx_V tv = gx_unboxed(target);
    gx_V cur = err;
    for (;;) {
        const gx_TypeDesc *ct = gx_dyn_type(cur);
        if (!ct) return gx_bool(false);
        gx_V cv = gx_unboxed(cur);
        if (tt->comparable && ct == tt && ct->eq(cv, tv)) return gx_bool(true);
        const gx_Method *is = gx_method(ct, "Is");
        if (is) {
            gx_V a[2] = {cv, target};
            if (gx_b(is->code(NULL, a, 2))) return gx_bool(true);
        }
        const gx_Method *unwrap = gx_method(ct, "Unwrap");
        if (!unwrap) return gx_bool(false);
        gx_V a[1] = {cv};
        cur = unwrap->code(NULL, a, 1);
    }
}
