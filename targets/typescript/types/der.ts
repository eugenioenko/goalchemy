// Bounded DER framing only. Cryptographic validation is performed by WebCrypto.
import { invalid, b64, unb64 } from './native.ts';
export interface DER { tag:number; raw:Uint8Array<ArrayBuffer>; body:Uint8Array<ArrayBuffer>; }
export function derRead(b:Uint8Array<ArrayBuffer>, start=0): [DER,number] {
  if(start+2>b.length)invalid();const tag=b[start++];const first=b[start++];let n=first;
  if(first&128){const count=first&127;if(count===0||count>4||start+count>b.length||b[start]===0)invalid();n=0;for(let i=0;i<count;i++)n=n*256+b[start++];if(n<128)invalid();}
  if(n>b.length-start)invalid();const end=start+n;return [{tag,raw:b.slice(start-(first&128?(first&127)+2:2),end),body:b.slice(start,end)},end];
}
export function children(b:Uint8Array<ArrayBuffer>): DER[]{const out:DER[]=[];let p=0;while(p<b.length){if(out.length>32)invalid();const [d,e]=derRead(b,p);out.push(d);p=e;}return out;}
export function sequence(b:Uint8Array<ArrayBuffer>): DER[]{const[d,e]=derRead(b);if(d.tag!==48||e!==b.length)invalid();return children(d.body);}
export function der(tag:number,...parts:Uint8Array<ArrayBuffer>[]):Uint8Array<ArrayBuffer>{const n=parts.reduce((a,b)=>a+b.length,0);if(n>65536)invalid();const len:number[]=[];let v=n;if(n<128)len.push(n);else{while(v){len.unshift(v&255);v=Math.floor(v/256);}len.unshift(128|len.length);}const out=new Uint8Array(1+len.length+n);out[0]=tag;out.set(len,1);let p=1+len.length;for(const b of parts){out.set(b,p);p+=b.length;}return out;}
const RSA_ALG=Uint8Array.from([48,13,6,9,42,134,72,134,247,13,1,1,1,5,0]);
export function parsePEM(pem:string):{format:'spki'|'pkcs8';data:Uint8Array<ArrayBuffer>}{
 if(pem.length>65536)invalid();const m=/^\s*-----BEGIN (PUBLIC KEY|PRIVATE KEY|RSA PUBLIC KEY|RSA PRIVATE KEY|CERTIFICATE)-----\r?\n([A-Za-z0-9+/=\r\n]+)\s*-----END \1-----\s*$/.exec(pem);if(!m)invalid();let b=unb64(m[2].replace(/[\r\n]/g,''),false,65536);let format:'spki'|'pkcs8'='spki';
 if(m[1]==='PRIVATE KEY')format='pkcs8';
 if(m[1]==='RSA PUBLIC KEY')b=der(48,RSA_ALG,der(3,new Uint8Array([0]),b));
 if(m[1]==='RSA PRIVATE KEY'){b=der(48,new Uint8Array([2,1,0]),RSA_ALG,der(4,b));format='pkcs8';}
 if(m[1]==='CERTIFICATE'){const cert=sequence(b);if(cert.length!==3)invalid();const tbs=sequence(cert[0].raw);const index=tbs[0]?.tag===160?6:5;if(tbs.length<=index)invalid();b=tbs[index].raw;}
 namedCurve(b,format);return {format,data:b};
}
const EC_ALG=Uint8Array.from([6,7,42,134,72,206,61,2,1]);
// WebCrypto maps explicit EC domain parameters onto a named curve without checking all of them.
function namedCurve(b:Uint8Array<ArrayBuffer>,format:'spki'|'pkcs8'):void{const parts=sequence(b);const id=parts[format==='spki'?0:1];if(id?.tag!==48)invalid();const alg=children(id.body);if(alg[0]&&alg[0].raw.length===EC_ALG.length&&alg[0].raw.every((x,i)=>x===EC_ALG[i])&&alg[1]?.tag!==6)invalid();}
export function pem(format:'spki'|'pkcs8',b:Uint8Array):string{const label=format==='spki'?'PUBLIC KEY':'PRIVATE KEY';const s=b64(b);return '-----BEGIN '+label+'-----\n'+s.match(/.{1,64}/g)!.join('\n')+'\n-----END '+label+'-----\n';}
