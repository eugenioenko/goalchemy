#include "native.h"
void gx_lib_crypto_rs256_verify(gx_Task *t,gx_V a0,gx_V a1,gx_V a2) { gx_crypto_call(t,"rs256_verify",3,(gx_V[]){a0,a1,a2},gx_bool(false)); }
