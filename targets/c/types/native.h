#ifndef GX_NATIVE_H
#define GX_NATIVE_H
#include "gx.h"
#include "library.h"
#define GX_NATIVE_MAX (64u*1024u*1024u)
extern const gxc_options *gxc_active_options;
void gx_crypto_call(gx_Task *t,const char *operation,int count,const gx_V *arguments,gx_V zero);
gx_V gx_native_encoding(gx_V v,bool url,bool decode);
#endif
