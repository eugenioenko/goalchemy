// Independent Node native cross-library oracle; production has no Node imports.
import { createPublicKey,createPrivateKey,publicEncrypt,privateDecrypt,constants,sign,verify,diffieHellman,createHmac,hkdfSync } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { mkdtempSync,writeFileSync,rmSync } from 'node:fs';
function certificateFor(pem:string):string{const dir=mkdtempSync('out/typescript-tdf-library/cert-node-');try{const file=dir+'/key.pem';writeFileSync(file,pem,{mode:0o600});return execFileSync('openssl',['req','-new','-x509','-key',file,'-subj','/CN=Goalchemy-native-test','-days','1'],{encoding:'utf8'});}finally{rmSync(dir,{recursive:true});}}
import * as c from '../types/crypto.ts';
import { runCryptoSuite,assert,hx } from './crypto_suite.ts';
for(const name of await runCryptoSuite())console.log('PASS',name);
const bytes=Uint8Array.from([0,255,128,7]);
for(const family of ['RSA','EC'] as const){
 const key=await c.generate(family),lease=key.acquire();const pem=await c.privatePEM(lease),publicPem=await c.publicPEM(lease),priv=createPrivateKey(pem),pub=createPublicKey(publicPem);
 const signature=await c.sign(family==='EC',lease,bytes);assert(verify('sha256',bytes,{key:pub,dsaEncoding:'ieee-p1363'},signature),'Node verifies WebCrypto');const nodeSig=sign('sha256',bytes,{key:priv,dsaEncoding:'ieee-p1363'});assert(await c.verify(family==='EC',lease,bytes,nodeSig),'WebCrypto verifies Node');
 const certificate=certificateFor(pem);const certified=await c.importPEM(certificate),certLease=certified.acquire();assert(await c.verify(family==='EC',certLease,bytes,signature),'independent OpenSSL certificate SPKI '+family);certLease.release();await certified.close();
 if(family==='RSA'){const encrypted=publicEncrypt({key:pub,padding:constants.RSA_PKCS1_OAEP_PADDING,oaepHash:'sha1'},bytes);assert(hx(await c.rsa(true,lease,encrypted))===hx(bytes),'Node SHA1 OAEP -> WebCrypto');const web=await c.rsa(false,lease,bytes);assert(hx(privateDecrypt({key:priv,padding:constants.RSA_PKCS1_OAEP_PADDING,oaepHash:'sha1'},web))===hx(bytes),'WebCrypto -> Node SHA1 OAEP');
 for(const type of ['pkcs1','spki'] as const){const imported=await c.importPEM(pub.export({format:'pem',type}).toString()),l=imported.acquire();assert(await c.verify(false,l,bytes,nodeSig),'RSA public PEM '+type);l.release();await imported.close();}const imported=await c.importPEM(priv.export({format:'pem',type:'pkcs1'}).toString()),l=imported.acquire();assert(hx(await c.rsa(true,l,encrypted))===hx(bytes),'RSA PKCS1 private PEM');l.release();await imported.close();
 }else{const other=await c.generate('EC'),ol=other.acquire();const opub=createPublicKey(await c.publicPEM(ol));assert(hx(await c.ecdh(lease,ol))===hx(diffieHellman({privateKey:priv,publicKey:opub})),'Node ECDH oracle');ol.release();await other.close();}
 lease.release();await key.close();
}
assert(hx(await c.hmac(bytes,bytes))===hx(createHmac('sha256',bytes).update(bytes).digest()),'Node HMAC oracle');assert(hx(await c.hkdf(bytes,bytes,bytes,42n))===hx(new Uint8Array(hkdfSync('sha256',bytes,bytes,bytes,42))),'Node HKDF oracle');
console.log('PASS independent Node OAEP/RS256/ES256/ECDH/HMAC/HKDF and PKCS1/SPKI/PKCS8');
