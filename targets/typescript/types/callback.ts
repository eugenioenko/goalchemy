/** Explicit settlement snapshots synchronously and acknowledges released resources. */
export type Settlement=(reply:Uint8Array|null,error?:unknown)=>void;
/** Promise fulfillment is snapshotted when observed. Explicit callbacks return
 * a nonblocking stop hook and must settle after resources retire, even on cancel. */
export type Callback=(signal:AbortSignal,request:Uint8Array|null,settle:Settlement)=>Promise<Uint8Array|null>|(()=>void)|void;
export let callbacks:Readonly<Record<string,Callback>>=Object.freeze({});
export function installCallbacks(value:Readonly<Record<string,Callback>>):void{callbacks=value;}
