#include "native.h"
void gx_lib_crypto_random(gx_Task *t,gx_V a0) { gx_crypto_call(t,"random",1,(gx_V[]){a0},gx_nil_byte_slice()); }
