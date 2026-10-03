#include "native.h"
void gx_lib_crypto_rsa_oaep_encrypt(gx_Task *t,gx_V a0,gx_V a1) { gx_crypto_call(t,"rsa_oaep_encrypt",2,(gx_V[]){a0,a1},gx_nil_byte_slice()); }
