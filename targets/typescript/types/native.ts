// Owned portable capability wire. Only the driver decodes source values.
import { Slice, BYTE_NIL, NIL } from './slice.ts';
import { stdErrorsNew } from '../runtime/std_errors_new.ts';
import type { Box } from './iface.ts';
export const MAX_BYTES = 64 << 20;
export class DeclaredFailure extends Error {}
export function invalid(message = 'crypto: invalid input or key'): never { throw new DeclaredFailure(message); }
export function bytes(s: Slice<number>, max = MAX_BYTES): Uint8Array<ArrayBuffer> {
  if (s.l < 0 || s.l > max) invalid();
  const b = new Uint8Array(s.l);
  for (let i=0;i<s.l;i++) b[i]=s.a![s.o+i];
  return b;
}
export function byteSlice(b: Uint8Array | null): Slice<number> { return b === null ? BYTE_NIL : new Slice(b,0,b.length,b.length,true); }
export function strings(s: Slice<string>): string[] { return s.a === null ? [] : Array.from({length:s.l},(_,i)=>s.a![s.o+i]); }
export function stringSlice(a: string[] | null): Slice<string> { return a === null ? NIL : new Slice(a,0,a.length,a.length); }
export function errorBox(message: string | null): Box | null { return message === null ? null : stdErrorsNew(message); }
export function binary(b: Uint8Array): string {
  let s=''; for(let i=0;i<b.length;i+=8192)s+=String.fromCharCode(...b.subarray(i,i+8192)); return s;
}
export function binaryInput(s: string, max = MAX_BYTES): Uint8Array<ArrayBuffer> {
  if(typeof s!=='string'||s.length>max) invalid();
  const b=new Uint8Array(s.length);for(let i=0;i<s.length;i++){const c=s.charCodeAt(i);if(c>255)invalid();b[i]=c;}return b;
}
export function b64(b: Uint8Array, url=false): string {
  const s=btoa(binary(b));return url?s.replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/,''):s;
}
export function unb64(s: string, url=false, max=MAX_BYTES): Uint8Array<ArrayBuffer> {
  if(typeof s!=='string'||s.length>Math.ceil(max/3)*4||!(url?/^[A-Za-z0-9_-]*$/:/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/).test(s))invalid('encoding: invalid input or size');
  let raw:string;try{raw=atob(url?s.replace(/-/g,'+').replace(/_/g,'/'):s);}catch{invalid('encoding: invalid input or size');}
  const b=binaryInput(raw,max);if(b64(b,url)!==s)invalid('encoding: invalid input or size');return b;
}
export function bounded(...xs: Uint8Array[]): void { if(xs.some(x=>x.length>MAX_BYTES))invalid(); }
export function size(n: bigint|number, max=MAX_BYTES): number {
  if(typeof n==='bigint'){if(n<0n||n>BigInt(max))invalid();return Number(n);}
  if(!Number.isSafeInteger(n)||n<0||n>max)invalid();return n;
}
