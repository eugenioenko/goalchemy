#include "native.h"
void gx_lib_crypto_rs256_sign(gx_Task *t,gx_V a0,gx_V a1) { gx_crypto_call(t,"rs256_sign",2,(gx_V[]){a0,a1},gx_nil_byte_slice()); }
