import { Frame, HostFault, sched, type Scheduler, type Task } from './task_spawn.ts';
import * as native from '../types/crypto.ts';
import { bytes, byteSlice, errorBox, stringSlice, DeclaredFailure, invalid } from '../types/native.ts';
import type { Slice } from '../types/slice.ts';
let nextKey=0;
/** Source opaque handle. Aliases share this object and its native lifetime. */
export class Key {
  readonly owner:object;
  readonly id:number;
  readonly native:native.NativeKey;
  constructor(owner:Scheduler,key:native.NativeKey){this.owner=owner.domain;this.native=key;this.id=++nextKey;}
}
function lease(k:Key|null):ReturnType<native.NativeKey['acquire']>{if(k===null)invalid();if(k.owner!==sched.domain)throw new HostFault('foreign key owner');return k.native.acquire();}
export function libCryptoClose(t:Task,k:Key|null):void {
  if(k===null){t.rv=[];return;}if(k.owner!==sched.domain)throw new HostFault('foreign key owner');
  const pending=k.native.close();const owner=sched;const token=owner.registerHost(t,()=>{});owner.launchHost(token,async()=>{await pending;return [];});
}
type Kind='bytes'|'strings'|'string'|'bool'|'key';
export function cryptoOperation(t:Task,kind:Kind,work:()=>Promise<unknown>,leases:ReturnType<native.NativeKey['acquire']>[]=[],early?:unknown):void{
 const owner=sched;let staged:native.NativeKey|null=null;let transferred=false;
 const decode=(rv:readonly unknown[]):unknown[]=>{
  const e=errorBox(rv[1] as string|null);if(e!==null)return [kind==='string'?'':kind==='bool'?false:kind==='bytes'?byteSlice(null):kind==='strings'?stringSlice(null):null,e];
  let value=rv[0];if(kind==='key'){if(staged===null||rv[0]!==id)throw new HostFault('missing staged native key');value=new Key(owner,staged);transferred=true;owner.disposers.delete(dispose);owner.retained.add(dispose);}
  else if(kind==='bytes')value=byteSlice(value as Uint8Array);else if(kind==='strings')value=stringSlice(value as string[]);
  return [value,null];
 };
 const id=++nextKey;
 const dispose=()=>{if(staged!==null){void staged.close();staged=null;}owner.disposers.delete(dispose);owner.retained.delete(dispose);};
 if(kind==='key')owner.disposers.add(dispose);
 const token=owner.registerHost(t,()=>{},()=>{if(kind==='key'&&!transferred)dispose();},()=>null,decode);
 owner.launchHost(token,async()=>{
  try{if(early!==undefined)throw early;const out=await work();if(kind==='key'){staged=out as native.NativeKey;if(owner.closed){await staged.close();staged=null;}return [id,null];}return [out,null];}
  catch(e){const message=native.declaredCryptoError(e);if(message!==null)return [null,message];throw e;}
  finally{for(const l of leases)l.release();}
 });
}
export function keyOperation(t:Task,kind:Kind,keys:(Key|null)[],work:(leases:ReturnType<native.NativeKey['acquire']>[])=>Promise<unknown>):void{
 const ls:ReturnType<native.NativeKey['acquire']>[]=[];try{for(const k of keys)ls.push(lease(k));}catch(e){for(const l of ls)l.release();if(!(e instanceof DeclaredFailure))throw e;cryptoOperation(t,kind,async()=>null,[],e);return;}
 cryptoOperation(t,kind,()=>work(ls),ls);
}
export function byteOperation(t:Task,kind:Kind,input:Slice<number>[],work:(b:Uint8Array<ArrayBuffer>[])=>Promise<unknown>,max?:number):void{
 try{const b=input.map(x=>bytes(x,max));cryptoOperation(t,kind,()=>work(b));}catch(e){if(!(e instanceof DeclaredFailure))throw e;cryptoOperation(t,kind,async()=>null,[],e);}
}
