package rt;

import java.math.BigInteger;
import java.nio.charset.StandardCharsets;
import java.security.*;
import java.security.interfaces.*;
import java.security.spec.*;
import javax.crypto.*;
import javax.crypto.spec.*;
import java.util.Arrays;
import java.util.Base64;

/** Native JCA primitives; pinned BC public APIs supply HKDF and derive-Q only. */
public final class Crypto {
    private Crypto() {}
    private static final SecureRandom RANDOM=new SecureRandom();
    private static final String INVALID="crypto: invalid input or key";
    private static final OAEPParameterSpec OAEP=new OAEPParameterSpec("SHA-1","MGF1",MGF1ParameterSpec.SHA1,PSource.PSpecified.DEFAULT);
    public static Native.Reject invalid(){return new Native.Reject(INVALID);}
    public static void bounded(byte[]... values)throws Native.Reject {for(byte[] value:values)if(value.length>Native.MAX)throw invalid();}
    public static int size(long n,int max)throws Native.Reject {if(n<0||n>max)throw invalid();return (int)n;}
    public static byte[] random(long n)throws Native.Reject {byte[] result=new byte[size(n,Native.MAX)];RANDOM.nextBytes(result);return result;}
    public static byte[] sha(byte[] data)throws Exception {bounded(data);return MessageDigest.getInstance("SHA-256").digest(data);}
    private static final class MACKey implements SecretKey {
        private final byte[] encoded;MACKey(byte[] input){encoded=input.clone();}
        public String getAlgorithm(){return "HmacSHA256";}public String getFormat(){return "RAW";}public byte[] getEncoded(){return encoded.clone();}
    }
    public static byte[] hmac(byte[] key,byte[] data)throws Exception {bounded(key,data);Mac m=Mac.getInstance("HmacSHA256");m.init(new MACKey(key));return m.doFinal(data);}
    public static boolean hmacVerify(byte[] key,byte[] data,byte[] expected)throws Exception {if(expected.length!=32)throw invalid();return MessageDigest.isEqual(hmac(key,data),expected);}
    private static Object invoke(java.lang.reflect.Method method,Object target,Object... args)throws Exception {
        try{return method.invoke(target,args);}catch(java.lang.reflect.InvocationTargetException e){throw new TaskSpawn.HostFault("BC native API failed: "+e.getCause());}
    }
    public static byte[] hkdf(byte[] secret,byte[] salt,byte[] info,long n)throws Exception {
        bounded(secret,salt,info);int length=size(n,8160);if(length==0)return new byte[0];
        // Exact pinned public BC API; missing linkage is an adapter fault, never reduced crypto.
        Class<?> digestClass=Class.forName("org.bouncycastle.crypto.Digest");
        Object digest=Class.forName("org.bouncycastle.crypto.digests.SHA256Digest").getConstructor().newInstance();
        Class<?> generatorClass=Class.forName("org.bouncycastle.crypto.generators.HKDFBytesGenerator");Object generator=generatorClass.getConstructor(digestClass).newInstance(digest);
        Object parameters=Class.forName("org.bouncycastle.crypto.params.HKDFParameters").getConstructor(byte[].class,byte[].class,byte[].class).newInstance(secret,salt,info);
        invoke(generatorClass.getMethod("init",Class.forName("org.bouncycastle.crypto.DerivationParameters")),generator,parameters);
        byte[] result=new byte[length];invoke(generatorClass.getMethod("generateBytes",byte[].class,int.class,int.class),generator,result,0,length);return result;
    }
    public static byte[] aes(byte[] key,byte[] nonce,byte[] data,byte[] aad,boolean encrypt)throws Exception {
        if(key.length!=32||nonce.length!=12||aad.length>Native.MAX||data.length>(encrypt?Native.MAX:Native.MAX+16)||!encrypt&&data.length<16)throw invalid();
        Cipher cipher=Cipher.getInstance("AES/GCM/NoPadding");cipher.init(encrypt?Cipher.ENCRYPT_MODE:Cipher.DECRYPT_MODE,new SecretKeySpec(key,"AES"),new GCMParameterSpec(128,nonce));cipher.updateAAD(aad);
        try{return cipher.doFinal(data);}catch(AEADBadTagException e){throw new Native.Reject("crypto: authentication failed");}
    }
    private static final String[] CURVES={"secp256r1","secp384r1","secp521r1"};
    private static ECParameterSpec params(String name)throws Exception {AlgorithmParameters p=AlgorithmParameters.getInstance("EC");p.init(new ECGenParameterSpec(name));return p.getParameterSpec(ECParameterSpec.class);}
    private static String curve(ECParameterSpec p)throws Exception {
        for(String name:CURVES){ECParameterSpec expected=params(name);if(p.getCurve().equals(expected.getCurve())&&p.getGenerator().equals(expected.getGenerator())&&p.getOrder().equals(expected.getOrder())&&p.getCofactor()==expected.getCofactor())return name;}
        return null;
    }
    private static int width(String curve){return curve.equals("secp256r1")?32:curve.equals("secp384r1")?48:66;}
    private static int width(ECKey key)throws Exception {String name=curve(key.getParams());if(name==null)throw invalid();return width(name);}
    public static Native.Material generate(String kind)throws Exception {
        boolean rsa=kind.startsWith("rsa");KeyPairGenerator g=KeyPairGenerator.getInstance(rsa?"RSA":"EC");
        if(rsa)g.initialize(Integer.parseInt(kind.substring(3)),RANDOM);else g.initialize(new ECGenParameterSpec(kind.equals("p256")?"secp256r1":kind.equals("p384")?"secp384r1":"secp521r1"),RANDOM);
        KeyPair pair=g.generateKeyPair();return new Native.Material(pair.getPrivate(),pair.getPublic());
    }
    private static PublicKey derive(ECPrivateKey privateKey)throws Exception {
        String name=curve(privateKey.getParams());if(name==null)throw invalid();ECParameterSpec params=params(name);BigInteger d=privateKey.getS();if(d.signum()<=0||d.compareTo(params.getOrder())>=0)throw invalid();
        Class<?> named=Class.forName("org.bouncycastle.asn1.sec.SECNamedCurves");Object namedParams=invoke(named.getMethod("getByName",String.class),null,name);
        Class<?> domain=Class.forName("org.bouncycastle.asn1.x9.X9ECParameters"),pointClass=Class.forName("org.bouncycastle.math.ec.ECPoint"),fieldClass=Class.forName("org.bouncycastle.math.ec.ECFieldElement");
        Object generator=invoke(domain.getMethod("getG"),namedParams);Class<?> multiplierClass=Class.forName("org.bouncycastle.math.ec.FixedPointCombMultiplier");Object multiplier=multiplierClass.getConstructor().newInstance();Object point=invoke(multiplierClass.getMethod("multiply",pointClass,BigInteger.class),multiplier,generator,d);point=invoke(pointClass.getMethod("normalize"),point);
        BigInteger x=(BigInteger)invoke(fieldClass.getMethod("toBigInteger"),invoke(pointClass.getMethod("getAffineXCoord"),point));
        BigInteger y=(BigInteger)invoke(fieldClass.getMethod("toBigInteger"),invoke(pointClass.getMethod("getAffineYCoord"),point));
        return KeyFactory.getInstance("EC").generatePublic(new ECPublicKeySpec(new ECPoint(x,y),params));
    }
    private static void validate(PublicKey key)throws Exception {
        if(key instanceof RSAPublicKey r){int bits=r.getModulus().bitLength();if(bits!=2048&&bits!=3072&&bits!=4096||r.getPublicExponent().compareTo(BigInteger.valueOf(3))<0||!r.getPublicExponent().testBit(0))throw invalid();return;}
        if(key instanceof ECPublicKey e){String name=curve(e.getParams());if(name==null)throw invalid();
            // Native ECDH validates point membership/coordinates on the exact curve.
            KeyPairGenerator g=KeyPairGenerator.getInstance("EC");g.initialize(new ECGenParameterSpec(name),RANDOM);KeyAgreement agreement=KeyAgreement.getInstance("ECDH");agreement.init(g.generateKeyPair().getPrivate());try{agreement.doPhase(e,true);agreement.generateSecret();}catch(InvalidKeyException exception){throw invalid();}return;}
        throw invalid();
    }
    public static Native.Material importPEM(String binary)throws Exception {
        if(binary.length()>65536)throw invalid();String text=Native.utf8(binary).strip();
        if(!text.startsWith("-----BEGIN "))throw invalid();int end=text.indexOf("-----",11);if(end<0)throw invalid();String label=text.substring(11,end);String begin="-----BEGIN "+label+"-----",finish="-----END "+label+"-----";
        if(!(text.startsWith(begin+"\n")||text.startsWith(begin+"\r\n"))||!text.endsWith(finish)||text.indexOf("-----BEGIN ",1)>=0)throw invalid();
        String body=text.substring(begin.length(),text.length()-finish.length());if(!body.matches("[A-Za-z0-9+/=\\r\\n\\t ]*"))throw invalid();byte[] der;
        try{der=Base64.getDecoder().decode(body.replaceAll("[\\r\\n\\t ]",""));}catch(IllegalArgumentException e){throw invalid();}
        Der.one(der);PublicKey pub=null;PrivateKey privateKey=null;
        if(label.equals("RSA PUBLIC KEY")) {der=Der.seq(Der.RSA_ALGORITHM,Der.value(3,Der.join(new byte[]{0},der)));label="PUBLIC KEY";}
        if(label.equals("RSA PRIVATE KEY")) {der=Der.seq(Der.integer(BigInteger.ZERO),Der.RSA_ALGORITHM,Der.value(4,der));label="PRIVATE KEY";}
        try {
            switch(label) {
            case "CERTIFICATE":pub=java.security.cert.CertificateFactory.getInstance("X.509").generateCertificate(new java.io.ByteArrayInputStream(der)).getPublicKey();break;
            case "PUBLIC KEY":
                try{pub=KeyFactory.getInstance("RSA").generatePublic(new X509EncodedKeySpec(der));}
                catch(InvalidKeySpecException e){pub=KeyFactory.getInstance("EC").generatePublic(new X509EncodedKeySpec(der));}break;
            case "PRIVATE KEY":
                try{privateKey=KeyFactory.getInstance("RSA").generatePrivate(new PKCS8EncodedKeySpec(der));}
                catch(InvalidKeySpecException e){privateKey=KeyFactory.getInstance("EC").generatePrivate(new PKCS8EncodedKeySpec(der));}
                if(privateKey instanceof RSAPrivateCrtKey r){pub=KeyFactory.getInstance("RSA").generatePublic(new RSAPublicKeySpec(r.getModulus(),r.getPublicExponent()));}
                else if(privateKey instanceof ECPrivateKey e){pub=derive(e);byte[] included=Der.includedPoint(der);if(included!=null&&!MessageDigest.isEqual(included,point((ECPublicKey)pub)))throw invalid();}
                else throw invalid();break;
            default:throw invalid();
            }
            validate(pub);
            if(privateKey!=null) {
                Signature proof=Signature.getInstance(pub instanceof RSAPublicKey?"SHA256withRSA":"SHA256withECDSA");proof.initSign(privateKey,RANDOM);proof.update(new byte[]{1,2,3});byte[] signature=proof.sign();proof.initVerify(pub);proof.update(new byte[]{1,2,3});if(!proof.verify(signature))throw invalid();
            }
            return new Native.Material(privateKey,pub);
        }catch(InvalidKeySpecException|InvalidKeyException|SignatureException|java.security.cert.CertificateException e){throw invalid();}
    }
    private static byte[] unsigned(BigInteger n){byte[] bytes=n.toByteArray();return bytes.length>1&&bytes[0]==0?Arrays.copyOfRange(bytes,1,bytes.length):bytes;}
    private static byte[] coordinate(BigInteger n,int width)throws Native.Reject {byte[] a=unsigned(n);if(a.length>width)throw invalid();byte[] r=new byte[width];System.arraycopy(a,0,r,width-a.length,a.length);return r;}
    private static byte[] point(ECPublicKey publicKey)throws Exception {int n=width(publicKey);return Der.join(new byte[]{4},coordinate(publicKey.getW().getAffineX(),n),coordinate(publicKey.getW().getAffineY(),n));}
    public static String publicPEM(Native.Lease key)throws Exception {if(key.publicKey==null)throw invalid();return pem("PUBLIC KEY",key.publicKey.getEncoded());}
    public static String privatePEM(Native.Lease key)throws Exception {if(key.privateKey==null)throw invalid();byte[] der=key.privateKey.getEncoded();if(key.publicKey instanceof ECPublicKey e)der=Der.privateWithPoint(der,point(e));return pem("PRIVATE KEY",der);}
    private static String pem(String label,byte[] data){return "-----BEGIN "+label+"-----\n"+Base64.getMimeEncoder(64,new byte[]{10}).encodeToString(data)+"\n-----END "+label+"-----\n";}
    public static String[] jwk(Native.Lease key)throws Exception {
        Base64.Encoder e=Base64.getUrlEncoder().withoutPadding();if(key.publicKey instanceof RSAPublicKey r)return new String[]{"RSA","",e.encodeToString(unsigned(r.getModulus())),e.encodeToString(unsigned(r.getPublicExponent())),"",""};
        if(key.publicKey instanceof ECPublicKey p){int n=width(p);return new String[]{"EC",n==32?"P-256":n==48?"P-384":"P-521","","",e.encodeToString(coordinate(p.getW().getAffineX(),n)),e.encodeToString(coordinate(p.getW().getAffineY(),n))};}throw invalid();
    }
    public static byte[] rsa(Native.Lease key,byte[] data,boolean encrypt)throws Exception {
        if(!(key.publicKey instanceof RSAPublicKey r))throw invalid();int n=(r.getModulus().bitLength()+7)/8;
        if(encrypt&&data.length>n-42||!encrypt&&data.length!=n||!encrypt&&!(key.privateKey instanceof RSAPrivateKey))throw invalid();
        Cipher cipher=Cipher.getInstance("RSA/ECB/OAEPWithSHA-1AndMGF1Padding");cipher.init(encrypt?Cipher.ENCRYPT_MODE:Cipher.DECRYPT_MODE,encrypt?key.publicKey:key.privateKey,OAEP,RANDOM);
        try{return cipher.doFinal(data);}catch(BadPaddingException|IllegalBlockSizeException e){throw new Native.Reject("crypto: decryption failed");}
    }
    /** JOSE algorithm such as "rs384" or "es512": the signature width in bytes, checking the key family and EC curve. */
    private static int signatureWidth(Native.Lease key,String alg)throws Exception {
        if(alg.startsWith("rs")){if(!(key.publicKey instanceof RSAPublicKey r))throw invalid();return (r.getModulus().bitLength()+7)/8;}
        if(!(key.publicKey instanceof ECPublicKey e))throw invalid();int n=width(e);if(n!=(alg.equals("es256")?32:alg.equals("es384")?48:66))throw invalid();return 2*n;
    }
    private static String signatureAlgorithm(String alg){return "SHA"+alg.substring(2)+(alg.startsWith("es")?"withECDSA":"withRSA");}
    public static byte[] sign(Native.Lease key,byte[] data,String alg)throws Exception {
        bounded(data);int n=signatureWidth(key,alg);boolean ec=alg.startsWith("es");if(ec?!(key.privateKey instanceof ECPrivateKey):!(key.privateKey instanceof RSAPrivateKey))throw invalid();
        Signature signature=Signature.getInstance(signatureAlgorithm(alg));signature.initSign(key.privateKey,RANDOM);signature.update(data);byte[] out=signature.sign();return ec?Der.rawSignature(out,n/2):out;
    }
    public static boolean verify(Native.Lease key,byte[] data,byte[] signature,String alg)throws Exception {
        bounded(data);int n=signatureWidth(key,alg);boolean ec=alg.startsWith("es");if(signature.length!=n)throw invalid();
        Signature verifier=Signature.getInstance(signatureAlgorithm(alg));verifier.initVerify(key.publicKey);verifier.update(data);try{return verifier.verify(ec?Der.derSignature(signature):signature);}catch(SignatureException e){return false;}
    }
    public static byte[] ecdh(Native.Lease privateKey,Native.Lease publicKey)throws Exception {
        if(!(privateKey.privateKey instanceof ECPrivateKey own)||!(publicKey.publicKey instanceof ECPublicKey peer))throw invalid();int n=width(own);if(n!=width(peer))throw invalid();
        KeyAgreement ecdh=KeyAgreement.getInstance("ECDH");ecdh.init(own);ecdh.doPhase(peer,true);byte[] result=ecdh.generateSecret();if(result.length!=n)throw new TaskSpawn.HostFault("native ECDH coordinate width");return result;
    }
    public static void execute(TaskSpawn.Task task,String operation,Native.Key[] keys,Object[] arguments,Object[] zero,String output) {
        Native.Lease[] leases=new Native.Lease[keys.length];boolean submitted=false;
        try {
            for(int i=0;i<keys.length;i++)leases[i]=Native.acquire(keys[i]);
            Object[] owned=arguments.clone();
            for(int i=0;i<owned.length;i++)if(owned[i] instanceof Slice slice)owned[i]=Native.bytes(slice,operation.equals("aes_decrypt")&&i==2?Native.MAX+16:Native.MAX);
            Native.State state=Native.state();
            Native.async(task,zero,()-> {
                Object value=switch(operation) {
                    case "random" -> random((Long)owned[0]);
                    case "sha" -> sha((byte[])owned[0]);
                    case "hmac" -> hmac((byte[])owned[0],(byte[])owned[1]);
                    case "hmac_verify" -> hmacVerify((byte[])owned[0],(byte[])owned[1],(byte[])owned[2]);
                    case "hkdf" -> hkdf((byte[])owned[0],(byte[])owned[1],(byte[])owned[2],(Long)owned[3]);
                    case "aes_encrypt", "aes_decrypt" -> aes((byte[])owned[0],(byte[])owned[1],(byte[])owned[2],(byte[])owned[3],operation.equals("aes_encrypt"));
                    case "rsa_encrypt", "rsa_decrypt" -> rsa(leases[0],(byte[])owned[0],operation.equals("rsa_encrypt"));
                    case "rs256_sign", "rs384_sign", "rs512_sign", "es256_sign", "es384_sign", "es512_sign" -> sign(leases[0],(byte[])owned[0],operation.substring(0,5));
                    case "rs256_verify", "rs384_verify", "rs512_verify", "es256_verify", "es384_verify", "es512_verify" -> verify(leases[0],(byte[])owned[0],(byte[])owned[1],operation.substring(0,5));
                    case "ecdh" -> ecdh(leases[0],leases[1]);
                    case "public_pem" -> publicPEM(leases[0]);
                    case "private_pem" -> privatePEM(leases[0]);
                    case "jwk" -> jwk(leases[0]);
                    case "generate_rsa2048", "generate_rsa4096", "generate_p256", "generate_p384", "generate_p521" -> state.stage(generate(operation.substring(9)));
                    case "import" -> state.stage(importPEM((String)owned[0]));
                    default -> throw new TaskSpawn.HostFault("unknown native crypto operation");
                };
                return new Object[]{value};
            },wire-> {
                try {
                    Object value=switch(output) {
                        case "bytes" -> Native.slice((byte[])wire[0]);
                        case "strings" -> Native.strings((String[])wire[0]);
                        case "key" -> Native.transfer(state,(Long)wire[0]);
                        default -> wire[0];
                    };
                    return new Object[]{value,null};
                }catch(Native.Reject e){return Native.failure(zero,e.getMessage());}
            },leases);
            submitted=true;
        }catch(Native.Reject e){Native.reject(task,zero,e);}
        finally {if(!submitted)for(Native.Lease lease:leases)if(lease!=null)lease.close();}
    }
}
