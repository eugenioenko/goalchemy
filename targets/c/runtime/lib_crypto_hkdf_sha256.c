#include "native.h"
void gx_lib_crypto_hkdf_sha256(gx_Task *t,gx_V a0,gx_V a1,gx_V a2,gx_V a3) { gx_crypto_call(t,"hkdf_sha256",4,(gx_V[]){a0,a1,a2,a3},gx_nil_byte_slice()); }
