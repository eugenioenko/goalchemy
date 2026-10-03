#include "native.h"
void gx_lib_crypto_aes256_gcm_decrypt(gx_Task *t,gx_V a0,gx_V a1,gx_V a2,gx_V a3) { gx_crypto_call(t,"aes256_gcm_decrypt",4,(gx_V[]){a0,a1,a2,a3},gx_nil_byte_slice()); }
