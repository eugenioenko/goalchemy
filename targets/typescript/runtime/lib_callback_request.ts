import { sched,HostFault,type Task } from './task_spawn.ts';
import { observeContext,onContextCancel,type Context } from './std_context_err.ts';
import { callbacks,type Settlement } from '../types/callback.ts';
import { bytes,byteSlice,errorBox } from '../types/native.ts';
import { BYTE_NIL,type Slice } from '../types/slice.ts';
export function libCallbackRequest(t:Task,ctx:Context|null,name:string,request:Slice<number>):void{
 if(ctx===null||!name||name.length>128||request.l>1048576){t.rv=[BYTE_NIL,errorBox('callback: invalid request')];return;}
 observeContext(ctx);if(ctx.err!==null){t.rv=[BYTE_NIL,ctx.err];return;}const cb=callbacks[name];if(!cb){t.rv=[BYTE_NIL,errorBox('callback: unavailable')];return;}
 const input=request.a===null?null:bytes(request,1048576),controller=new AbortController(),owner=sched;
 let hook:(()=>void)|null=null,stopped=false,cancelFault=false;
 const stop=()=>{if(stopped)return;stopped=true;controller.abort();try{hook?.();}catch{cancelFault=true;}};
 const unlink=onContextCancel(ctx,stop);
 const token=owner.registerHost(t,stop,unlink,()=>{observeContext(ctx);return ctx.err===null?null:[BYTE_NIL,ctx.err];},rv=>[byteSlice(rv[0] as Uint8Array|null),errorBox(rv[1] as string|null)]);
 owner.launchHost(token,async()=>{
  let resolve!:(rv:unknown[])=>void;const terminal=new Promise<unknown[]>(r=>resolve=r);let settled=false;
  const settle:Settlement=(reply,error)=>{
   if(settled)return;settled=true;
   if(error!==undefined&&error!==null){resolve([null,'callback: provider rejected']);return;}
   if(reply!==null&&(!(reply instanceof Uint8Array)||reply.length>1048576)){resolve([null,'callback: reply limit or shape']);return;}
   resolve([reply===null?null:new Uint8Array(reply),null]);
  };
  // Submission throws remain host faults. Only Promise rejection or explicit
  // settlement error is a declared provider rejection.
  const pending=cb(controller.signal,input,settle);
  if(typeof pending==='function'){hook=pending;if(stopped){try{hook();}catch{cancelFault=true;}}}
  else if(pending!==undefined){Promise.resolve(pending).then(reply=>settle(reply),()=>settle(null,new Error('provider rejected')));}
  const result=await terminal;
  if(cancelFault)throw new HostFault('callback cancellation hook fault');
  return result;
 });
}
