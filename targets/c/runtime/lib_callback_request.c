#include "native.h"
#include <stdlib.h>
typedef struct callback_work {gx_HostToken token;pthread_t thread;bool started;atomic_bool canceled;gxc_provider provider;void *state;uint8_t *name,*payload;size_t name_len,payload_len;} callback_work;
static void callback_zero(gx_Task *t,gx_V err){gx_V r[2]={gx_nil_byte_slice(),err};gx_set_rv(t,2,r);}
static const char *callback_cancel(void *arg){atomic_store(&((callback_work *)arg)->canceled,true);return NULL;}
static const char *callback_cleanup(void *arg){callback_work *w=arg;if(w->started&&pthread_join(w->thread,NULL))return "cannot join provider worker";free(w->name);free(w->payload);free(w);return NULL;}
static const char *callback_decode(gx_Task *t,const uint8_t *b,size_t n,gx_V cancel,gx_V roots){(void)roots;if(!gx_is_nil(cancel)){callback_zero(t,cancel);return NULL;}if(n<1||n>128*1024+1)return "provider wire bounds";if(b[0])callback_zero(t,gx_std_errors_new(gx_str((const char *)b+1,n-1)));else{uint8_t *data=gx_alloc_bytes(n-1);if(n>1)memcpy(data,b+1,n-1);gx_V r[2]={gx_byte_slice(data,(uint32_t)n-1,(uint32_t)n-1),gx_nil()};gx_set_rv(t,2,r);}return NULL;}
static void *callback_thread(void *arg){callback_work *w=arg;gxc_value response={0};gxc_callback_thread=true;int rc=w->provider(w->state,w->name,w->name_len,w->payload,w->payload_len,&w->canceled,&response);gxc_callback_thread=false;const char *fault=NULL;
 if(response.kind!=GXC_BYTES||response.length>128*1024||response.length&&!response.bytes){fault="provider invalid native response";}else if(rc!=0&&rc!=1)fault="provider native host fault";
 uint8_t *wire=NULL;size_t n=response.length+1;if(!fault){wire=malloc(n);if(!wire)fault="provider publication allocation";else{wire[0]=(uint8_t)rc;if(response.length)memcpy(wire+1,response.bytes,response.length);}}
 gxc_value_free(&response);free(w->name);w->name=NULL;free(w->payload);w->payload=NULL;w->provider=NULL;w->state=NULL;
 if(!gx_host_publish(w->token,wire,wire?n:0,fault))gx_host_publish_fault(w->token,"provider native publication failure");free(wire);gx_host_ack(w->token);gx_host_token_release(w->token);return NULL;}
void gx_lib_callback_request(gx_Task *t,gx_V ctx,gx_V name,gx_V payload){
 if(gx_is_nil(ctx)||name.t!=GX_STR||!name.l||name.l>256||payload.t!=GX_SLICE||!gx_byte_backing(payload)||payload.l>GX_NATIVE_MAX){callback_zero(t,gx_std_errors_new(gx_cstr("callback: invalid input")));return;}
 gx_HostBoundary boundary=gx_host_boundary(ctx,0);gx_V e=gx_std_context_context_err(ctx);if(!gx_is_nil(e)){callback_zero(t,e);return;}
 if(!gxc_active_options||!gxc_active_options->provider){callback_zero(t,gx_std_errors_new(gx_cstr("callback: provider not found")));return;}
 callback_work *w=calloc(1,sizeof(*w));if(!w)gx_host_fault("provider holder allocation");atomic_init(&w->canceled,false);w->provider=gxc_active_options->provider;w->state=gxc_active_options->provider_state;w->name_len=name.l;w->payload_len=payload.l;
 w->name=malloc(name.l);w->payload=malloc(payload.l?payload.l:1);if(!w->name||!w->payload){callback_cleanup(w);gx_host_fault("provider snapshot allocation");}memcpy(w->name,gx_sbytes(name),name.l);if(payload.l)memcpy(w->payload,gx_bytes(payload),payload.l);
 w->token=gx_host_register(t,boundary,ctx,callback_decode,callback_cancel,callback_cleanup,w);gx_host_token_retain(w->token);
 if(!gx_host_should_submit(w->token)){free(w->name);w->name=NULL;free(w->payload);w->payload=NULL;gx_host_publish(w->token,NULL,0,NULL);gx_host_ack(w->token);gx_host_token_release(w->token);return;}
 if(pthread_create(&w->thread,NULL,callback_thread,w)){free(w->name);w->name=NULL;free(w->payload);w->payload=NULL;gx_host_publish(w->token,NULL,0,"provider submission");gx_host_ack(w->token);gx_host_token_release(w->token);return;}w->started=true;
}
