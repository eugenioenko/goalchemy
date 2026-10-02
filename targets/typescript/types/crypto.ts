// Portable maintained WebCrypto primitives; no protocol construction.
import { invalid, bounded, size, DeclaredFailure, b64, unb64, MAX_BYTES } from './native.ts';
import { parsePEM, pem } from './der.ts';
export type KeyUse='oaep'|'sign'|'ecdh';
export class NativeKey {
  private jwk:JsonWebKey|null;
  private cache=new Map<string,Promise<CryptoKey>>();
  private leases=0;
  private closing=false;
  private waiters:(()=>void)[]=[];
  readonly family:'RSA'|'EC';
  constructor(jwk:JsonWebKey){validateJWK(jwk);this.jwk={...jwk};this.family=jwk.kty as 'RSA'|'EC';}
  acquire():{jwk:JsonWebKey; release:()=>void; key:(use:KeyUse,privateKey:boolean)=>Promise<CryptoKey>}{
    if(this.closing||this.jwk===null)invalid('crypto: key is closed');const jwk=this.jwk;this.leases++;let released=false;
    return {jwk,release:()=>{if(released)return;released=true;if(--this.leases===0&&this.closing)this.drop();},key:(use,priv)=>{
      if((priv&&!jwk.d)||(use==='ecdh'&&this.family!=='EC')||(use==='oaep'&&this.family!=='RSA'))invalid();
      const id=use+':'+priv;let p=this.cache.get(id);if(!p){const copy={...jwk};delete copy.alg;delete copy.key_ops;copy.ext=true;if(!priv)for(const n of ['d','p','q','dp','dq','qi'] as const)delete copy[n];
        const alg=this.family==='RSA'?{name:use==='oaep'?'RSA-OAEP':'RSASSA-PKCS1-v1_5',hash:use==='oaep'?'SHA-1':'SHA-256'}:{name:use==='ecdh'?'ECDH':'ECDSA',namedCurve:'P-256'};
        const uses:KeyUsage[]=use==='ecdh'?(priv?['deriveBits']:[]):use==='oaep'?(priv?['decrypt']:['encrypt']):(priv?['sign']:['verify']);
        p=crypto.subtle.importKey('jwk',copy,alg,true,uses);this.cache.set(id,p);
      }return p;
    }};
  }
  close():Promise<void>{this.closing=true;if(!this.leases){this.drop();return Promise.resolve();}return new Promise(resolve=>this.waiters.push(resolve));}
  private drop():void{this.jwk=null;this.cache.clear();for(const w of this.waiters)w();this.waiters=[];}
  // Tests observe resource state without copying native private material.
  state():{closed:boolean;leases:number;retained:boolean}{return {closed:this.closing,leases:this.leases,retained:this.jwk!==null};}
}
function validateJWK(j:JsonWebKey):void{
 if(j.kty==='RSA'){if(!j.n||!j.e)invalid();const n=unb64(j.n,true,256),e=unb64(j.e,true,8);if(n.length!==256||!(n[0]&128)||e.length===0)invalid();let v=0n;for(const b of e)v=v*256n+BigInt(b);if(v<3n||v>2147483647n||(v&1n)===0n)invalid();}
 else if(j.kty==='EC'){if(j.crv!=='P-256'||!j.x||!j.y||unb64(j.x,true,32).length!==32||unb64(j.y,true,32).length!==32||(j.d&&unb64(j.d,true,32).length!==32))invalid();}else invalid();
}
export async function generate(family:'RSA'|'EC'):Promise<NativeKey>{const pair=await crypto.subtle.generateKey(family==='RSA'?{name:'RSASSA-PKCS1-v1_5',modulusLength:2048,publicExponent:new Uint8Array([1,0,1]),hash:'SHA-256'}:{name:'ECDSA',namedCurve:'P-256'},true,['sign','verify']);return new NativeKey(await crypto.subtle.exportKey('jwk',pair.privateKey));}
export async function importPEM(s:string):Promise<NativeKey>{const p=parsePEM(s);for(const alg of [{name:'RSASSA-PKCS1-v1_5',hash:'SHA-256'},{name:'ECDSA',namedCurve:'P-256'}]){try{const key=await crypto.subtle.importKey(p.format,p.data,alg,true,[p.format==='pkcs8'?'sign':'verify']);return new NativeKey(await crypto.subtle.exportKey('jwk',key));}catch(e){if(e instanceof DeclaredFailure)throw e;if(!(e instanceof DOMException))throw e;}}invalid();}
export async function publicPEM(lease:ReturnType<NativeKey['acquire']>):Promise<string>{const key=await lease.key('sign',false);return pem('spki',new Uint8Array(await crypto.subtle.exportKey('spki',key)));}
export async function privatePEM(lease:ReturnType<NativeKey['acquire']>):Promise<string>{const key=await lease.key('sign',true);return pem('pkcs8',new Uint8Array(await crypto.subtle.exportKey('pkcs8',key)));}
export function publicJWK(lease:ReturnType<NativeKey['acquire']>):string[]{const j=lease.jwk;return j.kty==='RSA'?['RSA','',j.n!,j.e!,'','']:['EC','P-256','','',j.x!,j.y!];}
export async function digest(data:Uint8Array<ArrayBuffer>):Promise<Uint8Array<ArrayBuffer>>{bounded(data);return new Uint8Array(await crypto.subtle.digest('SHA-256',data));}
export function random(n:bigint|number):Uint8Array{const out=new Uint8Array(size(n));for(let i=0;i<out.length;i+=65536)crypto.getRandomValues(out.subarray(i,Math.min(i+65536,out.length)));return out;}
async function hmacKey(key:Uint8Array<ArrayBuffer>,use:KeyUsage):Promise<CryptoKey>{bounded(key);return crypto.subtle.importKey('raw',key.length?key:new Uint8Array(64),{name:'HMAC',hash:'SHA-256'},false,[use]);}
export async function hmac(key:Uint8Array<ArrayBuffer>,data:Uint8Array<ArrayBuffer>):Promise<Uint8Array<ArrayBuffer>>{bounded(key,data);return new Uint8Array(await crypto.subtle.sign('HMAC',await hmacKey(key,'sign'),data));}
export async function hmacVerify(key:Uint8Array<ArrayBuffer>,data:Uint8Array<ArrayBuffer>,mac:Uint8Array<ArrayBuffer>):Promise<boolean>{bounded(key,data);if(mac.length!==32)invalid();return crypto.subtle.verify('HMAC',await hmacKey(key,'verify'),mac,data);}
export async function hkdf(secret:Uint8Array<ArrayBuffer>,salt:Uint8Array<ArrayBuffer>,info:Uint8Array<ArrayBuffer>,n:bigint|number):Promise<Uint8Array<ArrayBuffer>>{bounded(secret,salt,info);const len=size(n,8160);if(!len)return new Uint8Array();const key=await crypto.subtle.importKey('raw',secret,'HKDF',false,['deriveBits']);return new Uint8Array(await crypto.subtle.deriveBits({name:'HKDF',hash:'SHA-256',salt,info},key,len*8));}
export async function aes(decrypt:boolean,key:Uint8Array<ArrayBuffer>,nonce:Uint8Array<ArrayBuffer>,data:Uint8Array<ArrayBuffer>,aad:Uint8Array<ArrayBuffer>):Promise<Uint8Array<ArrayBuffer>>{if(key.length!==32||nonce.length!==12||aad.length>MAX_BYTES||data.length>(decrypt?MAX_BYTES+16:MAX_BYTES)||(decrypt&&data.length<16))invalid();const use=decrypt?'decrypt':'encrypt';const k=await crypto.subtle.importKey('raw',key,'AES-GCM',false,[use]);return new Uint8Array(await crypto.subtle[use]({name:'AES-GCM',iv:nonce,additionalData:aad,tagLength:128},k,data));}
export async function rsa(decrypt:boolean,lease:ReturnType<NativeKey['acquire']>,data:Uint8Array<ArrayBuffer>):Promise<Uint8Array<ArrayBuffer>>{if(lease.jwk.kty!=='RSA'||(decrypt?data.length!==256:data.length>214))invalid();const k=await lease.key('oaep',decrypt);return new Uint8Array(await crypto.subtle[decrypt?'decrypt':'encrypt']({name:'RSA-OAEP',label:new Uint8Array()},k,data));}
export async function sign(ec:boolean,lease:ReturnType<NativeKey['acquire']>,data:Uint8Array<ArrayBuffer>):Promise<Uint8Array<ArrayBuffer>>{bounded(data);if(lease.jwk.kty!==(ec?'EC':'RSA'))invalid();const k=await lease.key('sign',true);const result=new Uint8Array(await crypto.subtle.sign(ec?{name:'ECDSA',hash:'SHA-256'}:'RSASSA-PKCS1-v1_5',k,data));if(result.length!==(ec?64:256))invalid();return result;}
export async function verify(ec:boolean,lease:ReturnType<NativeKey['acquire']>,data:Uint8Array<ArrayBuffer>,signature:Uint8Array<ArrayBuffer>):Promise<boolean>{bounded(data);if(lease.jwk.kty!==(ec?'EC':'RSA')||signature.length!==(ec?64:256))invalid();return crypto.subtle.verify(ec?{name:'ECDSA',hash:'SHA-256'}:'RSASSA-PKCS1-v1_5',await lease.key('sign',false),signature,data);}
export async function ecdh(a:ReturnType<NativeKey['acquire']>,b:ReturnType<NativeKey['acquire']>):Promise<Uint8Array<ArrayBuffer>>{const priv=await a.key('ecdh',true),pub=await b.key('ecdh',false);return new Uint8Array(await crypto.subtle.deriveBits({name:'ECDH',public:pub},priv,256));}
export function declaredCryptoError(e:unknown):string|null{if(e instanceof DeclaredFailure)return e.message;if(e instanceof DOMException&&['OperationError','DataError','InvalidAccessError','NotSupportedError','SyntaxError'].includes(e.name))return 'crypto: operation failed';return null;}
