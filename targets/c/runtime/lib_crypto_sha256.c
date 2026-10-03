#include "native.h"
void gx_lib_crypto_sha256(gx_Task *t,gx_V a0) { gx_crypto_call(t,"sha256",1,(gx_V[]){a0},gx_nil_byte_slice()); }
