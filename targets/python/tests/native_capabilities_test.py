"""Independent known-answer and native interoperability capability checks."""
import sys,pathlib,tempfile,subprocess,threading,base64,json,os
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parents[2]))
from python import rt_index as rt
from python.runtime import lib_crypto_close as native
from cryptography.hazmat.primitives.asymmetric import ec,rsa,utils
from cryptography.hazmat.primitives import serialization,hashes
checks=0

def check(value,label):
 global checks
 assert value,label;checks+=1

def data(v):return b'' if v.a is None else bytes(v.a[v.o:v.o+v.l])
def call(op,*args):
 t=rt.sched().cur;getattr(rt,'lib_crypto_'+op)(t,*args);return t.rv

def ok(op,*args):
 v,e=call(op,*args);check(e is None,'crypto success '+op);return v

def invalid(op,*args):
 v,e=call(op,*args);check(e is not None,'crypto rejection '+op);return v
b=native.slice_bytes

def vectors():
 backing=b(b'XabcY');view=rt.reslice(backing,1,4)
 saved=native.byte_input(view);rt.sset(backing,1,ord('z'))
 check(saved==b'abc' and native.byte_input(view)==b'zbc','native bytes snapshot respects offset and owns input')
 check(native.byte_input(rt.BYTE_NIL)==b'','native nil byte input')
 check(data(ok('sha256',b(b'abc'))).hex()=='ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad','SHA256')
 mac=ok('hmac_sha256',b(b'\x0b'*20),b(b'Hi There'));check(data(mac).hex()=='b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7','RFC4231')
 check(ok('hmac_sha256_verify',b(b'\x0b'*20),b(b'Hi There'),mac),'HMAC verify');check(not ok('hmac_sha256_verify',b(b'\x0b'*20),b(b'bad'),mac),'HMAC mismatch');invalid('hmac_sha256_verify',b(b''),b(b''),b(b''))
 out=ok('hkdf_sha256',b(b'\x0b'*22),b(bytes.fromhex('000102030405060708090a0b0c')),b(bytes.fromhex('f0f1f2f3f4f5f6f7f8f9')),42)
 check(data(out).hex()=='3cb25f25faacd57a90434f64d0362f2a2d2d0a90cf1a5a4c5db02d56ecc4c5bf34007208d5b887185865','RFC5869');check(data(ok('hkdf_sha256',b(b''),b(b''),b(b''),0))==b'','HKDF0');invalid('hkdf_sha256',b(b''),b(b''),b(b''),8161)
 cipher=ok('aes256gcm_encrypt',b(bytes(32)),b(bytes(12)),b(bytes(16)),b(b''))
 check(data(cipher).hex()=='cea7403d4d606b6e074ec5d3baf39d18d0d1c8a799996bf0265b98b5d48ab919','NIST AES256GCM');check(data(ok('aes256gcm_decrypt',b(bytes(32)),b(bytes(12)),cipher,b(b'')))==bytes(16),'AES decrypt')
 bad=bytearray(data(cipher));bad[-1]^=1;check(invalid('aes256gcm_decrypt',b(bytes(32)),b(bytes(12)),b(bad),b(b'')).a is None,'no partial plaintext');invalid('aes256gcm_encrypt',b(bytes(16)),b(bytes(12)),b(b''),b(b''));invalid('aes256gcm_encrypt',b(bytes(32)),b(bytes(11)),b(b''),b(b''))
 key=ok('generate_rsa2048');pem=ok('private_pem',key);pub=ok('public_pem',key);imported=ok('import_pem',pub)
 check(ok('public_pem',imported)==pub,'SPKI import');check(ok('private_pem',ok('import_pem',pem))==pem,'PKCS8 import');invalid('private_pem',imported)
 jwk=ok('public_jwk',key);check(jwk.a[0]==b'RSA' and len(jwk.a[2])==342 and jwk.a[3]==b'AQAB','RSA JWK')
 plain=b(b'\0\xffOAEP');cipher=ok('rsaoaep_encrypt',imported,plain);check(data(ok('rsaoaep_decrypt',key,cipher))==data(plain),'OAEP');invalid('rsaoaep_encrypt',key,b(bytes(215)));invalid('rsaoaep_decrypt',key,b(bytes(255)))
 sig=ok('rs256_sign',key,plain);check(ok('rs256_verify',imported,plain,sig),'RS256');check(not ok('rs256_verify',imported,b(b'bad'),sig),'RS256 mismatch');invalid('rs256_verify',key,plain,b(bytes(255)))
 with tempfile.TemporaryDirectory() as tmp:
  p=pathlib.Path(tmp);(p/'private.pem').write_bytes(pem);(p/'public.pem').write_bytes(pub);(p/'message').write_bytes(data(plain));(p/'cipher').write_bytes(data(cipher));(p/'sig').write_bytes(data(sig))
  def openssl(*args):return subprocess.run(['openssl',*args],check=True,stdout=subprocess.PIPE,stderr=subprocess.PIPE).stdout
  decrypted=openssl('pkeyutl','-decrypt','-inkey',str(p/'private.pem'),'-in',str(p/'cipher'),'-pkeyopt','rsa_padding_mode:oaep','-pkeyopt','rsa_oaep_md:sha1','-pkeyopt','rsa_mgf1_md:sha1');check(decrypted==data(plain),'independent OpenSSL OAEP SHA1/MGF1SHA1')
  openssl('dgst','-sha256','-verify',str(p/'public.pem'),'-signature',str(p/'sig'),str(p/'message'));check(True,'independent OpenSSL RS256')
  reverse=openssl('pkeyutl','-encrypt','-pubin','-inkey',str(p/'public.pem'),'-in',str(p/'message'),'-pkeyopt','rsa_padding_mode:oaep','-pkeyopt','rsa_oaep_md:sha1','-pkeyopt','rsa_mgf1_md:sha1');check(data(ok('rsaoaep_decrypt',key,b(reverse)))==data(plain),'OpenSSL-produced OAEP')
  openssl('req','-new','-x509','-key',str(p/'private.pem'),'-out',str(p/'cert.pem'),'-subj','/CN=fixture','-days','1');check(ok('public_pem',ok('import_pem',(p/'cert.pem').read_bytes()))==pub,'certificate public key')
 nativekey=serialization.load_pem_private_key(pem,password=None)
 pkcs1=nativekey.private_bytes(serialization.Encoding.PEM,serialization.PrivateFormat.TraditionalOpenSSL,serialization.NoEncryption());check(ok('public_pem',ok('import_pem',pkcs1))==pub,'PKCS1 private')
 pkcs1pub=nativekey.public_key().public_bytes(serialization.Encoding.PEM,serialization.PublicFormat.PKCS1);check(ok('public_pem',ok('import_pem',pkcs1pub))==pub,'PKCS1 public')
 for malformed in (b'',b'prefix'+pub,pub+pub,pub+b'x',pub.replace(b'PUBLIC KEY',b'RSA PUBLIC KEY'),b'-----BEGIN PRIVATE KEY-----\n!\n-----END PRIVATE KEY-----'):
  invalid('import_pem',malformed)
 invalid('import_pem',rsa.generate_private_key(public_exponent=65537,key_size=1024).public_key().public_bytes(serialization.Encoding.PEM,serialization.PublicFormat.SubjectPublicKeyInfo))
 invalid('import_pem',ec.generate_private_key(ec.SECP384R1()).public_key().public_bytes(serialization.Encoding.PEM,serialization.PublicFormat.SubjectPublicKeyInfo))
 def fixed_scalar(n):return ec.derive_private_key(n,ec.SECP256R1()).private_bytes(serialization.Encoding.PEM,serialization.PrivateFormat.PKCS8,serialization.NoEncryption())
 one=ok('import_pem',fixed_scalar(1));two=ok('import_pem',fixed_scalar(2));shared=ok('ecdh',one,two)
 # RFC5915's publicKey field is optional inside PKCS8. This fixed synthetic
 # scalar1 fixture omits it; every operation still uses maintained native crypto.
 omitted_der=bytes.fromhex('3041020100301306072a8648ce3d020106082a8648ce3d030107042730250201010420'+'00'*31+'01')
 omitted_pem=b'-----BEGIN PRIVATE KEY-----\n'+base64.encodebytes(omitted_der)+b'-----END PRIVATE KEY-----\n'
 omitted=ok('import_pem',omitted_pem)
 check(ok('public_pem',omitted)==ok('public_pem',one),'omitted-Q public parity')
 omitted_sig=ok('es256_sign',omitted,b(b'omitted-Q'));check(ok('es256_verify',one,b(b'omitted-Q'),omitted_sig),'omitted-Q sign/verify')
 check(data(ok('ecdh',omitted,two))==data(shared),'omitted-Q ECDH32')
 with tempfile.TemporaryDirectory() as tmp:
  p=pathlib.Path(tmp);(p/'omitted.pem').write_bytes(omitted_pem);(p/'peer.pem').write_bytes(fixed_scalar(2));(p/'python.sig').write_bytes(data(omitted_sig));(p/'public.pem').write_bytes(ok('public_pem',omitted));(p/'secret').write_bytes(data(shared))
  root=pathlib.Path(__file__).resolve().parents[3]
  (p/'go.mod').write_text('module pkcs8nativeprobe\n\ngo 1.25\nrequire github.com/eugenioenko/goalchemy v0.0.0\nreplace github.com/eugenioenko/goalchemy => '+str(root)+'\n')
  (p/'main.go').write_bytes((pathlib.Path(__file__).parent/'pkcs8_native_probe.go.in').read_bytes())
  env=dict(os.environ,GOTOOLCHAIN='go1.25.14')
  proof=subprocess.run(['go','run','.'],cwd=p,env=env,stdout=subprocess.PIPE,stderr=subprocess.PIPE,timeout=30)
  check(proof.returncode==0,'native Go omitted-Q import/public/ES256/ECDH '+proof.stderr.decode())
  check(ok('es256_verify',omitted,b(b'omitted-Q'),b((p/'go.sig').read_bytes())),'native Go signature consumed by Python')
  print(proof.stdout.decode().strip())
 for bad_der in (omitted_der+b'\0',omitted_der+bytes.fromhex('3000')):
  invalid('import_pem',b'-----BEGIN PRIVATE KEY-----\n'+base64.encodebytes(bad_der)+b'-----END PRIVATE KEY-----\n')
 for wrong in (pkcs1.replace(b'RSA PRIVATE KEY',b'PRIVATE KEY'),ec.derive_private_key(1,ec.SECP256R1()).private_bytes(serialization.Encoding.PEM,serialization.PrivateFormat.TraditionalOpenSSL,serialization.NoEncryption()).replace(b'EC PRIVATE KEY',b'PRIVATE KEY'),pem.replace(b'PRIVATE KEY',b'RSA PRIVATE KEY')):
  invalid('import_pem',wrong)

 check(data(shared).hex()=='7cf27b188d034f7e8a52380304b51ac3c08969e277f21b35a60b48fc47669978','P256 ECDH fixed scalar x(2G)');check(data(ok('ecdh',two,one))==data(shared),'ECDH both directions');check(len(data(ok('ecdh',one,one)))==32,'aliased ECDH')
 sig=ok('es256_sign',one,plain);check(len(data(sig))==64 and ok('es256_verify',one,plain,sig),'ES256 P1363');check(not ok('es256_verify',one,b(b'bad'),sig),'ES256 mismatch');invalid('es256_verify',one,plain,b(bytes(63)))
 r=int.from_bytes(data(sig)[:32],'big');s=int.from_bytes(data(sig)[32:],'big');ec.derive_private_key(1,ec.SECP256R1()).public_key().verify(utils.encode_dss_signature(r,s),data(plain),ec.ECDSA(hashes.SHA256()));check(True,'independent DER verifier')
 jwk=ok('public_jwk',one);check(jwk.a[0]==b'EC' and jwk.a[1]==b'P-256' and len(jwk.a[4])==43,'EC JWK')
 alias=one;rt.lib_crypto_close(rt.sched().cur,one);invalid('public_pem',alias);rt.lib_crypto_close(rt.sched().cur,one);rt.lib_crypto_close(rt.sched().cur,None);check(True,'idempotent aliases close')
 invalid('public_pem',None);invalid('random',-1);invalid('random',native.MAX+1);check(len(data(ok('random',32)))==32,'random length')
 for v in (b'',b'\0\xff',bytes(range(256))):
  encoded,e=rt.lib_encoding_base64_encode(b(v));check(e is None,'base64 encoding');decoded,e=rt.lib_encoding_base64_decode(encoded);check(e is None and data(decoded)==v,'base64');encoded,e=rt.lib_encoding_base64_url_encode(b(v));check(e is None,'base64url encoding');decoded,e=rt.lib_encoding_base64_url_decode(encoded);check(e is None and data(decoded)==v,'base64url')
 for malformed in (b'AA',b'AA===',b'AB==',b'AA==\n',b'AA-_'):
  v,e=rt.lib_encoding_base64_decode(malformed);check(e is not None and v.a is None,'strict base64')
 for malformed in (b'A',b'AB',b'AA=',b'AA+',b'AA\n'):
  v,e=rt.lib_encoding_base64_url_decode(malformed);check(e is not None and v.a is None,'strict base64url')
 native.native_retire(rt.sched());check(not rt.sched().native_keys,'native registry release')
 return []
rt.run_main_host(lambda:rt.sync(vectors))
print('PASS Python native capability checks',checks)
