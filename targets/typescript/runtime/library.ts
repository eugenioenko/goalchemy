// Reusable serialized, cancellable value-library boundary.
import { Frame, call, ret, driveLibrary, FatalPanic, HostFault, NativeCanceled, sync, type Task, type Scheduler } from './task_spawn.ts';
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
 return runLibraryCall(options,{reset},[],ctx=>entry(ctx),(_inst,rv)=>capture(rv));
}

/** Boundary warnings such as leaked handles and abandoned goroutines. */
export let libraryWarn:(message:string)=>void=message=>{console.warn('goalchemy: warning: '+message);};
export function setLibraryWarn(fn:(message:string)=>void):void{libraryWarn=fn;}

/** Saves, restores and zeroes the generated source globals. */
export interface LibraryState { reset:()=>void; save?:()=>unknown; load?:(saved:unknown)=>void; }

/** Persistent source state behind a family of handles; only touched while
 * holding the library reservation. */
export class Instance {
 globals:unknown=undefined;
 readonly domain:object={};
 readonly objects=new Map<unknown,{handle:Handle;ref:WeakRef<object>}>();
 readonly retain:(()=>void)[]=[];
 live=0;poisoned=false;retired=false;
 retireNow():void{if(this.retired)return;this.retired=true;this.globals=undefined;this.objects.clear();for(const r of this.retain.splice(0)){try{r();}catch{}}}
}

/** The host's reference to one source object of an instance. */
export class Handle {
 readonly inst:Instance;obj:unknown;readonly name:string;
 closed=false;released=false;
 readonly closing=new AbortController();
 active:AbortController|null=null;
 constructor(inst:Instance,obj:unknown,name:string){this.inst=inst;this.obj=obj;this.name=name;}
 release():void{if(this.released)return;this.released=true;const e=this.inst.objects.get(this.obj);if(e?.handle===this)this.inst.objects.delete(this.obj);this.inst.live--;this.obj=null;if(this.inst.live===0)this.inst.retireNow();}
}

const finalizers=new FinalizationRegistry<Handle>(h=>{
 if(h.closed)return;h.closed=true;
 void acquire().then(unlock=>{try{if(!h.released){libraryWarn(h.name+' handle was not closed; releasing it without running Close');h.release();}}finally{unlock();}});
});

/** Returns the host wrapper for obj, reusing the live wrapper of the same
 * source object so host identity follows source identity. */
export function wrap<W extends object>(inst:Instance,obj:unknown,name:string,make:(h:Handle)=>W):W|null{
 if(obj===null||obj===undefined)return null;
 const e=inst.objects.get(obj);if(e&&!e.handle.closed){const w=e.ref.deref();if(w)return w as W;}
 const h=new Handle(inst,obj,name),w=make(h);
 inst.objects.set(obj,{handle:h,ref:new WeakRef(w)});inst.live++;finalizers.register(w,h,h);return w;
}
export function obj<T>(h:Handle|null):T{return (h===null?null:h.obj) as T;}

export async function runLibraryCall(options:CallOptions,state:LibraryState,handles:(Handle|null)[],entry:(ctx:Context,fresh:boolean)=>Frame,capture:(inst:Instance,rv:unknown[])=>unknown[],closing:Handle|null=null):Promise<unknown[]>{
 const self=handles.find(h=>h!==null)??null;
 if(self!==null&&self!==closing&&self.closed)throw new LibraryError('closed');
 let signal=options.signal;let callAbort:AbortController|null=null;
 if(self!==null&&self!==closing){callAbort=new AbortController();const parts=[self.closing.signal,callAbort.signal];if(signal)parts.push(signal);signal=AbortSignal.any(parts);}
 let unlock:()=>void;
 try{unlock=await acquire(signal);}catch(e){if(self!==null&&self!==closing&&self.closing.signal.aborted)throw new LibraryError('closed');throw e;}
 let inst:Instance|null=null;
 try{
  for(const h of handles){if(h===null)continue;if(h.released||(h!==closing&&h.closed))throw new LibraryError('closed');if(inst!==null&&h.inst!==inst)throw new LibraryError('instance_mismatch');inst=h.inst;}
 }catch(e){unlock();throw e;}
 const fresh=inst===null;const owner=inst??new Instance();
 if(!fresh&&owner.poisoned){unlock();throw new LibraryError('poisoned');}
 if(self!==null&&self!==closing)self.active=callAbort;
 let cancel=()=>{};let keep=!fresh;
 try{
  installCallbacks(options.callbacks??Object.freeze({}));
  if(!fresh)state.load?.(owner.globals);
  return await driveLibrary(()=>{if(fresh)state.reset();const[ctx,stop]=stdContextWithCancel(BACKGROUND);cancel=stop;return entry(ctx,fresh);},rv=>{const out=capture(owner,rv);if(fresh&&owner.live>0)keep=true;return out;},signal,()=>cancel(),{domain:owner.domain,retire:(s:Scheduler,failure:unknown)=>{
   if(failure instanceof FatalPanic||(failure!==undefined&&!(failure instanceof NativeCanceled)&&!(failure instanceof LibraryError))){if(!fresh)owner.poisoned=true;return;}
   if(!keep)return;
   let left=0;for(const t of s.tasks)if(t!==s.main&&!t.done)left++;
   if(left>0)libraryWarn(left+' goroutine(s) still running when a handle call returned were abandoned');
   if(state.save)owner.globals=state.save();
   owner.retain.push(...s.retained);s.retained.clear();
  }});
 }catch(e){
  if(e instanceof NativeCanceled){if(self!==null&&self!==closing&&self.closing.signal.aborted)throw new LibraryError('canceled',{},'closed');throw new LibraryError('canceled',{},options.signal?.reason);}
  if(e instanceof LibraryError)throw e;if(e instanceof FatalPanic)throw new LibraryError('source_panic');if(e instanceof HostFault)throw new LibraryError('host_fault');throw new LibraryError('host_fault');
 }
 finally{
  state.reset();installCallbacks(Object.freeze({}));
  if(self!==null&&self!==closing)self.active=null;
  if(closing!==null)closing.release();
  else if(keep&&owner.live===0)owner.retireNow();
  unlock();
 }
}

/** Fails calls queued on h, cancels its active call, runs the source Close
 * when run is given and the instance is healthy, then releases h. */
export async function closeHandle(h:Handle,state:LibraryState,run:((ctx:Context,fresh:boolean)=>Frame)|null,failure:(err:unknown)=>LibraryError|null):Promise<void>{
 if(h.closed)return;h.closed=true;h.closing.abort();h.active?.abort();
 const entry=run??(()=>sync(()=>[null]));
 let rv:unknown[];
 try{rv=await runLibraryCall({},state,[h],entry,(_inst,rv)=>[failure(rv[0])],h);}
 catch(e){
  if(e instanceof LibraryError&&e.kind==='poisoned'){const unlock=await acquire();try{h.release();}finally{unlock();}return;}
  throw e;
 }
 if(rv[0]!==null)throw rv[0];
}

/** Source float boundary accepts numeric IEEE values, including NaN and infinities. */
export function boundaryFloat(v:unknown,bits:number):number {
 if(v===undefined)return 0;
 if(typeof v!=='number')invalidBoundary();
 return bits===32?Math.fround(v as number):v as number;
}
