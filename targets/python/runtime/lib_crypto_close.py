"""Native cryptography capability and owner key registry; no custom crypto."""
import base64
import binascii
import re
import os
import time
from cryptography import x509
from cryptography.exceptions import InvalidSignature, InvalidTag, UnsupportedAlgorithm
from cryptography.hazmat.primitives import hashes, serialization, hmac
from cryptography.hazmat.primitives.asymmetric import rsa, ec, padding, utils
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
from cryptography.hazmat.primitives.kdf.hkdf import HKDF
from .task_spawn import sched, HostFault
from .std_errors_new import std_errors_new
from ..types.slice import Slice, BYTE_NIL, NIL

MAX=64<<20
class Reject(Exception): pass
class Key:
    __slots__=('owner','identity')
    def __init__(self,owner,identity): self.owner=owner;self.identity=identity

def native_begin(callbacks):
    s=sched();s.native_keys={};s.native_next_key=0;s.library_callbacks=callbacks;return s

def native_retire(s):
    s.check();s.native_keys.clear();s.library_callbacks={}

def material(k):
    s=sched()
    if type(k) is not Key or k.owner is not s: raise Reject('crypto: invalid input or key')
    if k.identity not in s.native_keys: raise Reject('crypto: key is closed')
    return s.native_keys[k.identity]

def key(v):
    s=sched()
    if not hasattr(s,'native_keys'): native_begin({})
    s.native_next_key+=1;s.native_keys[s.native_next_key]=v
    return Key(s,s.native_next_key)

def byte_input(v,limit=MAX):
    if type(v) is not Slice or v.l>limit: raise Reject('crypto: invalid input or key')
    return b'' if v.a is None else bytes(memoryview(v.a)[v.o:v.o+v.l])

def slice_bytes(v):
    a=bytearray(v);return Slice(a,0,len(a),len(a),True)

def slice_strings(v): return Slice(list(v),0,len(v),len(v))

def string_input(v,limit=MAX):
    if type(v) is not bytes or len(v)>limit: raise Reject('crypto: invalid input or key')
    return v

def public(k): return k.public_key() if hasattr(k,'private_bytes') else k

def valid(k):
    if isinstance(k,(rsa.RSAPrivateKey,rsa.RSAPublicKey)):
        p=public(k).public_numbers()
        return k.key_size==2048 and p.e>=3 and p.e%2==1
    return isinstance(k,(ec.EllipticCurvePrivateKey,ec.EllipticCurvePublicKey)) and isinstance(k.curve,ec.SECP256R1)

def require(k,cls):
    if not isinstance(k,cls): raise Reject('crypto: invalid input or key')
    return k

def crypto_value(op,args):
    if op=='random':
        n=args[0]
        if type(n) is not int or n<0 or n>MAX: raise Reject('crypto: invalid input or key')
        return os.urandom(n)
    if op=='generate_rsa2048': return rsa.generate_private_key(public_exponent=65537,key_size=2048)
    if op=='generate_p256': return ec.generate_private_key(ec.SECP256R1())
    if op=='import_pem':
        data=string_input(args[0],65536).strip()
        m=re.fullmatch(rb'-----BEGIN (PUBLIC KEY|PRIVATE KEY|RSA PUBLIC KEY|RSA PRIVATE KEY|CERTIFICATE)-----\r?\n([A-Za-z0-9+/=\r\n]+)-----END \1-----',data)
        if m is None: raise Reject('crypto: invalid input or key')
        # DER loaders must consume the exactly matching PEM kind.
        der=base64.b64decode(m[2].replace(b'\r',b'').replace(b'\n',b''),validate=True)
        tag=m[1]
        if tag==b'CERTIFICATE': k=x509.load_der_x509_certificate(der).public_key()
        elif tag in (b'PUBLIC KEY',b'RSA PUBLIC KEY'): k=serialization.load_der_public_key(der)
        else:k=serialization.load_pem_private_key(data,password=None)
        if not valid(k): raise Reject('crypto: invalid input or key')
        # Require encoding shape to correspond to its advertised PEM type.
        fmt=serialization.PublicFormat.PKCS1 if tag==b'RSA PUBLIC KEY' else serialization.PublicFormat.SubjectPublicKeyInfo
        if tag in (b'PUBLIC KEY',b'RSA PUBLIC KEY') and k.public_bytes(serialization.Encoding.DER,fmt)!=der: raise Reject('crypto: invalid input or key')
        # Maintained PEM parsing enforces PKCS8/PKCS1 labels and full DER
        # consumption while accepting optional PKCS8/EC fields (e.g. omitted Q).
        if tag==b'RSA PRIVATE KEY' and not isinstance(k,rsa.RSAPrivateKey):raise Reject('crypto: invalid input or key')
        return k
    if op in ('sha256','hmac_sha256','hmac_sha256_verify','hkdf_sha256','aes256gcm_encrypt','aes256gcm_decrypt'):
        b=[byte_input(x,MAX+16 if op=='aes256gcm_decrypt' and i==2 else MAX) for i,x in enumerate(args) if not (op=='hkdf_sha256' and i==3)]
        if op=='sha256':
            h=hashes.Hash(hashes.SHA256());h.update(b[0]);return h.finalize()
        if op.startswith('hmac'):
            h=hmac.HMAC(b[0],hashes.SHA256());h.update(b[1])
            if op=='hmac_sha256':return h.finalize()
            if len(b[2])!=32:raise Reject('crypto: invalid input or key')
            try:h.verify(b[2]);return True
            except InvalidSignature:return False
        if op=='hkdf_sha256':
            n=args[3]
            if type(n) is not int or n<0 or n>8160:raise Reject('crypto: invalid input or key')
            return HKDF(algorithm=hashes.SHA256(),length=n,salt=b[1],info=b[2]).derive(b[0])
        if len(b[0])!=32 or len(b[1])!=12 or op.endswith('decrypt') and len(b[2])<16:raise Reject('crypto: invalid input or key')
        g=AESGCM(b[0]);return g.decrypt(b[1],b[2],b[3]) if op.endswith('decrypt') else g.encrypt(b[1],b[2],b[3])
    k=material(args[0])
    if op=='public_pem':return public(k).public_bytes(serialization.Encoding.PEM,serialization.PublicFormat.SubjectPublicKeyInfo)
    if op=='private_pem':
        if not isinstance(k,(rsa.RSAPrivateKey,ec.EllipticCurvePrivateKey)):raise Reject('crypto: invalid input or key')
        return k.private_bytes(serialization.Encoding.PEM,serialization.PrivateFormat.PKCS8,serialization.NoEncryption())
    if op=='public_jwk':
        p=public(k).public_numbers()
        enc=lambda n,width=None:base64.urlsafe_b64encode(n.to_bytes(width or (n.bit_length()+7)//8,'big')).rstrip(b'=')
        if isinstance(public(k),rsa.RSAPublicKey):return [b'RSA',b'',enc(p.n),enc(p.e),b'',b'']
        return [b'EC',b'P-256',b'',b'',enc(p.x,32),enc(p.y,32)]
    if op=='ecdh':return require(k,ec.EllipticCurvePrivateKey).exchange(ec.ECDH(),require(public(material(args[1])),ec.EllipticCurvePublicKey))
    data=byte_input(args[1]);sig=byte_input(args[2]) if len(args)>2 else None
    if op=='rsaoaep_encrypt':
        if len(data)>214:raise Reject('crypto: invalid input or key')
        return require(public(k),rsa.RSAPublicKey).encrypt(data,padding.OAEP(mgf=padding.MGF1(hashes.SHA1()),algorithm=hashes.SHA1(),label=None))
    if op=='rsaoaep_decrypt':
        if len(data)!=256:raise Reject('crypto: invalid input or key')
        return require(k,rsa.RSAPrivateKey).decrypt(data,padding.OAEP(mgf=padding.MGF1(hashes.SHA1()),algorithm=hashes.SHA1(),label=None))
    if op=='rs256_sign':return require(k,rsa.RSAPrivateKey).sign(data,padding.PKCS1v15(),hashes.SHA256())
    if op=='es256_sign':
        r,s=utils.decode_dss_signature(require(k,ec.EllipticCurvePrivateKey).sign(data,ec.ECDSA(hashes.SHA256())))
        return r.to_bytes(32,'big')+s.to_bytes(32,'big')
    if op=='rs256_verify':
        if len(sig)!=256:raise Reject('crypto: invalid input or key')
        verifier=require(public(k),rsa.RSAPublicKey);parameters=(padding.PKCS1v15(),hashes.SHA256())
    elif op=='es256_verify':
        if len(sig)!=64:raise Reject('crypto: invalid input or key')
        sig=utils.encode_dss_signature(int.from_bytes(sig[:32],'big'),int.from_bytes(sig[32:],'big'));verifier=require(public(k),ec.EllipticCurvePublicKey);parameters=(ec.ECDSA(hashes.SHA256()),)
    else:raise HostFault('unknown crypto operation')
    try:verifier.verify(sig,data,*parameters);return True
    except InvalidSignature:return False

def crypto_call(t,op,args,kind):
    sched().check()
    zero={'bytes':BYTE_NIL,'key':None,'bool':False,'text':b'','strings':NIL}[kind]
    try:
        v=crypto_value(op,args)
        if kind=='bytes':v=slice_bytes(v)
        elif kind=='key':v=key(v)
        elif kind=='strings':v=slice_strings(v)
        t.rv=[v,None]
    except (Reject,ValueError,InvalidTag,UnsupportedAlgorithm,binascii.Error) as e:
        message=str(e) if type(e) is Reject else 'crypto: invalid input or key'
        t.rv=[zero,std_errors_new(message.encode())]
    except BaseException as e:raise HostFault('crypto adapter fault') from e

def lib_crypto_close(t,k):
    sched().check()
    if k is not None:
        if type(k) is not Key or k.owner is not sched():raise HostFault('foreign native key close')
        sched().native_keys.pop(k.identity,None)
    t.rv=[]
