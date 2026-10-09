#include "native.h"
void gx_lib_crypto_es384_sign(gx_Task *t,gx_V a0,gx_V a1) { gx_crypto_call(t,"es384_sign",2,(gx_V[]){a0,a1},gx_nil_byte_slice()); }
