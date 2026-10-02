/* Canonical value encoding for the C harness. */
#ifndef GX_CODEC_H
#define GX_CODEC_H

#include "gx.h"
#include "json.h"

typedef gx_V (*DecFn)(J *);
typedef J *(*EncFn)(gx_V);

typedef struct H {
    J *lets;
    J *after;
} H;

J *h_let(H *h, const char *n);
void h_after(H *h, const char *n, J *v);

gx_V dec_int(J *r);
gx_V dec_bool(J *r);
gx_V dec_string(J *r);
gx_V dec_error(J *r);
gx_V dec_nil(J *r);
gx_V dec_slice(J *r, DecFn dec, gx_ZeroFn zero, bool bytes);
gx_V view(gx_V base, J *r);
gx_V dec_map(J *r, DecFn dk, DecFn dv);
gx_V dec_chan(J *r, DecFn dec, gx_ZeroFn zero);

J *enc_int_s(gx_V v);
J *enc_int_u(gx_V v);
J *enc_bool(gx_V v);
J *enc_string(gx_V v);
J *enc_error(gx_V v);
J *enc_zero(gx_V v);
J *enc_slice(gx_V v, EncFn enc);
J *enc_array(gx_V v, size_t n, EncFn enc);
J *enc_map(gx_V v, EncFn ek, EncFn ev);
J *enc_chan(gx_V v, EncFn enc);

gx_V new_spawn_check(void);
void harness_select2(gx_Task *t, gx_V a, gx_V b, gx_V dflt);

typedef J *(*CaseFn)(H *h);
typedef struct Case {
    const char *name;
    CaseFn fn;
} Case;
extern const Case CASES[];
extern const size_t NCASES;

#endif
