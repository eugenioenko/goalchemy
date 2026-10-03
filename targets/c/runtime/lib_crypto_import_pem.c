#include "native.h"
void gx_lib_crypto_import_pem(gx_Task *t,gx_V a0) { gx_crypto_call(t,"import_pem",1,(gx_V[]){a0},gx_nil()); }
