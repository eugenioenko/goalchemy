// Reusable serialized, cancellable value-library boundary.
import { Frame, call, ret, driveLibrary, FatalPanic, HostFault, NativeCanceled, type Task } from './task_spawn.ts';
import { stdContextWithCancel } from './std_context_with_cancel.ts';
import { BACKGROUND, type Context } from './std_context_err.ts';
import { installCallbacks, type Callback } from '../types/callback.ts';
export type { Callback, Settlement } from '../types/callback.ts';
export interface CallOptions { signal?:AbortSignal; callbacks?:Readonly<Record<string,Callback>>; }
export class LibraryError extends Error {
 readonly kind:string;
 readonly fields:Readonly<Record<string,unknown>>;
 constructor(kind:string,fields:Record<string,unknown>={},cause?:unknown){super('goalchemy library: '+kind,cause===undefined?undefined:{cause});this.name='LibraryError';this.kind=kind;this.fields=fields;}
}
export function invalidBoundary():never{throw new LibraryError('invalid_argument');}
export function boundaryInt(v:unknown,bits:number,signed:boolean):bigint|number{
 const wide=bits===64;if(v===undefined)return wide?0n:0;
 if(wide?typeof v!=='bigint':typeof v!=='number'||!Number.isInteger(v))invalidBoundary();
 const n=wide?v as bigint:BigInt(v as number),lo=signed?-(1n<<BigInt(bits-1)):0n,hi=(1n<<BigInt(signed?bits-1:bits))-1n;if(n<lo||n>hi)invalidBoundary();return wide?n:Number(n);
}
export function boundaryString(v:unknown):string{if(v===undefined)return '';if(typeof v!=='string'||v.length>64<<20)invalidBoundary();for(let i=0;i<v.length;i++)if(v.charCodeAt(i)>255)invalidBoundary();return v;}
export function boundaryBool(v:unknown):boolean{if(v===undefined)return false;if(typeof v!=='boolean')invalidBoundary();return v;}
export function boundaryObject(v:unknown):Record<string,unknown>{if(v===undefined)return {};if(v===null||typeof v!=='object'||Array.isArray(v))invalidBoundary();return v as Record<string,unknown>;}
export function boundaryArray(v:unknown):unknown[]|null{if(v===undefined||v===null)return null;if(!Array.isArray(v)||v.length>64<<20)invalidBoundary();return v;}
export function boundaryBytes(v:unknown):Uint8Array|null{if(v===undefined||v===null)return null;if(!(v instanceof Uint8Array)||v.length>64<<20)invalidBoundary();return new Uint8Array(v);}
interface Waiter { resolve:(release:()=>void)=>void;reject:(e:unknown)=>void;signal?:AbortSignal;abort:()=>void; }
let reserved=false;const queue:Waiter[]=[];
function release():void {for(;;){const w=queue.shift();if(!w){reserved=false;return;}w.signal?.removeEventListener('abort',w.abort);if(w.signal?.aborted){w.reject(new LibraryError('canceled',{},w.signal.reason));continue;}w.resolve(release);return;}}
function acquire(signal?:AbortSignal):Promise<()=>void>{if(signal?.aborted)return Promise.reject(new LibraryError('canceled',{},signal.reason));if(!reserved){reserved=true;return Promise.resolve(release);}return new Promise((resolve,reject)=>{const w:Waiter={resolve,reject,signal,abort:()=>{const i=queue.indexOf(w);if(i>=0){queue.splice(i,1);reject(new LibraryError('canceled',{},signal?.reason));}}};queue.push(w);signal?.addEventListener('abort',w.abort,{once:true});});}
export function snapshotOptions(options:CallOptions):CallOptions{
 if(options===null||typeof options!=='object'||(options.signal!==undefined&&!(options.signal instanceof AbortSignal)))invalidBoundary();const callbacks:Record<string,Callback>=Object.create(null);
 if(options.callbacks!==undefined){for(const [name,fn] of Object.entries(options.callbacks)){if(!name||name.length>128||typeof fn!=='function')invalidBoundary();callbacks[name]=fn;}}
 return {signal:options.signal,callbacks:Object.freeze(callbacks)};
}
class Sequence extends Frame {
 readonly init:Frame;readonly operation:()=>Frame;res:unknown[]=[];
 constructor(init:Frame,operation:()=>Frame){super();this.init=init;this.operation=operation;}
 step(t:Task):void{if(this.pc===0){this.pc=1;call(t,this.init);return;}if(this.pc===1){this.pc=2;call(t,this.operation());return;}this.res=t.rv;ret(t,this);}
 results():unknown[]{return this.res;}
}
export function librarySequence(init:Frame,operation:()=>Frame):Frame{return new Sequence(init,operation);}
export async function runLibrary(options:CallOptions,entry:(ctx:Context)=>Frame,reset:()=>void,capture:(rv:unknown[])=>unknown[]):Promise<unknown[]>{
 const unlock=await acquire(options.signal);let cancel=()=>{};
 try{
  installCallbacks(options.callbacks??Object.freeze({}));
  return await driveLibrary(()=>{reset();const[ctx,stop]=stdContextWithCancel(BACKGROUND);cancel=stop;return entry(ctx);},capture,options.signal,()=>cancel());
 }catch(e){if(e instanceof NativeCanceled)throw new LibraryError('canceled',{},options.signal?.reason);if(e instanceof LibraryError)throw e;if(e instanceof FatalPanic)throw new LibraryError('source_panic');if(e instanceof HostFault)throw new LibraryError('host_fault');throw new LibraryError('host_fault');}
 finally{reset();installCallbacks(Object.freeze({}));unlock();}
}
