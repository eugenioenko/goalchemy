package rt;
import java.nio.file.*;
import java.nio.charset.StandardCharsets;
import java.security.*;
import java.security.interfaces.*;
import java.security.spec.*;
import javax.crypto.*;
import javax.crypto.spec.*;
import java.util.*;
/** Published vectors and separately configured native JCA framing checks. */
public final class CryptoTest {
 static int checks;
 static byte[] h(String s){return HexFormat.of().parseHex(s);}
 static void check(boolean v,String name){checks++;if(!v)throw new AssertionError(name);}
 static void eq(byte[] a,String expected,String name){check(MessageDigest.isEqual(a,h(expected)),name);}
 interface Test {void run()throws Exception;}
 static void reject(Test test,String name)throws Exception{try{test.run();throw new AssertionError(name);}catch(Native.Reject e){checks++;}}
 static String pem(String type,byte[] b){return "-----BEGIN "+type+"-----\n"+Base64.getMimeEncoder(64,new byte[]{10}).encodeToString(b)+"\n-----END "+type+"-----\n";}
 public static void main(String[] args)throws Exception{
  eq(Crypto.sha(new byte[0]),"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","SHAempty");
  eq(Crypto.sha(h("616263")),"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad","SHAabc");
  eq(Crypto.hmac(new byte[0],new byte[0]),"b613679a0814d9ec772f95d778c35fc5ff1697c493715653c6c712144292c5ad","nativeemptyHMAC");
  byte[] key=new byte[20];Arrays.fill(key,(byte)11);byte[] data="Hi There".getBytes(StandardCharsets.US_ASCII);
  eq(Crypto.hmac(key,data),"b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7","RFC4231");
  byte[] mac=Crypto.hmac(key,data);check(Crypto.hmacVerify(key,data,mac),"verifyHMAC");mac[0]^=1;check(!Crypto.hmacVerify(key,data,mac),"rejectHMAC");reject(()->Crypto.hmacVerify(key,data,new byte[31]),"HMACsize");
  byte[] ikm=new byte[22];Arrays.fill(ikm,(byte)11);
  eq(Crypto.hkdf(ikm,h("000102030405060708090a0b0c"),h("f0f1f2f3f4f5f6f7f8f9"),42),"3cb25f25faacd57a90434f64d0362f2a2d2d0a90cf1a5a4c5db02d56ecc4c5bf34007208d5b887185865","RFC5869test1");
  eq(Crypto.hkdf(ikm,new byte[0],new byte[0],42),"8da4e775a563c18f715f802a063c5a31b8a11f5c5ee1879ec3454e5f3c738d2d9d201395faa4b61a96c8","RFC5869empty");
  check(Crypto.hkdf(ikm,new byte[0],new byte[0],0).length==0,"HKDF0");reject(()->Crypto.hkdf(ikm,new byte[0],new byte[0],8161),"HKDFbound");
  byte[] cipher=Crypto.aes(new byte[32],new byte[12],new byte[16],new byte[0],true);
  eq(cipher,"cea7403d4d606b6e074ec5d3baf39d18d0d1c8a799996bf0265b98b5d48ab919","NISTAES");check(Arrays.equals(Crypto.aes(new byte[32],new byte[12],cipher,new byte[0],false),new byte[16]),"AESdecrypt");cipher[0]^=1;reject(()->Crypto.aes(new byte[32],new byte[12],cipher,new byte[0],false),"AESreject");
  for(boolean rsa:new boolean[]{true,false}){
   Native.Material material=Crypto.generate(rsa?"rsa2048":"p256");
   try(Native.Lease own=material.acquire()){
    Native.Material imported=Crypto.importPEM(Crypto.privatePEM(own)),publicOnly=Crypto.importPEM(Crypto.publicPEM(own));
    try(Native.Lease a=imported.acquire();Native.Lease b=publicOnly.acquire()){
     check(Arrays.equals(own.publicKey.getEncoded(),a.publicKey.getEncoded()),"PEMidentity");check(Arrays.equals(Crypto.jwk(own),Crypto.jwk(a)),"JWKidentity");
     byte[] sig=Crypto.sign(a,data,rsa?"rs256":"es256");check(Crypto.verify(b,data,sig,rsa?"rs256":"es256"),"nativeJOSE");sig[0]^=1;check(!Crypto.verify(b,data,sig,rsa?"rs256":"es256"),"signatureReject");
     Signature independent=Signature.getInstance(rsa?"SHA256withRSA":"SHA256withECDSA");independent.initSign(own.privateKey);independent.update(data);byte[] nativeSig=independent.sign();check(Crypto.verify(b,data,rsa?nativeSig:Der.rawSignature(nativeSig,32),rsa?"rs256":"es256"),"JCAtoAdapter");
     independent.initVerify(own.publicKey);independent.update(data);byte[] adapterSig=Crypto.sign(a,data,rsa?"rs256":"es256");check(independent.verify(rsa?adapterSig:Der.derSignature(adapterSig)),"AdapterToJCA");
     if(rsa){
      Cipher oaep=Cipher.getInstance("RSA/ECB/OAEPPadding");OAEPParameterSpec spec=new OAEPParameterSpec("SHA-1","MGF1",MGF1ParameterSpec.SHA1,PSource.PSpecified.DEFAULT);oaep.init(Cipher.ENCRYPT_MODE,own.publicKey,spec);check(Arrays.equals(Crypto.rsa(a,oaep.doFinal(data),false),data),"JCAOAEPtoAdapter");oaep.init(Cipher.DECRYPT_MODE,own.privateKey,spec);check(Arrays.equals(oaep.doFinal(Crypto.rsa(b,data,true)),data),"AdapterOAEPtoJCA");reject(()->Crypto.rsa(b,new byte[215],true),"OAEPbound");
      RSAPublicKey publicKey=(RSAPublicKey)own.publicKey;Native.Material public1=Crypto.importPEM(pem("RSA PUBLIC KEY",Der.seq(Der.integer(publicKey.getModulus()),Der.integer(publicKey.getPublicExponent()))));try(Native.Lease p=public1.acquire()){check(Arrays.equals(Crypto.jwk(p),Crypto.jwk(own)),"PKCS1public");}finally{public1.close();}
      RSAPrivateCrtKey k=(RSAPrivateCrtKey)own.privateKey;byte[] pkcs1=Der.seq(Der.integer(java.math.BigInteger.ZERO),Der.integer(k.getModulus()),Der.integer(k.getPublicExponent()),Der.integer(k.getPrivateExponent()),Der.integer(k.getPrimeP()),Der.integer(k.getPrimeQ()),Der.integer(k.getPrimeExponentP()),Der.integer(k.getPrimeExponentQ()),Der.integer(k.getCrtCoefficient()));Native.Material p1=Crypto.importPEM(pem("RSA PRIVATE KEY",pkcs1));try(Native.Lease p=p1.acquire()){check(Arrays.equals(Crypto.jwk(p),Crypto.jwk(own)),"PKCS1private");}finally{p1.close();}
     }else{
      Native.Material other=Crypto.generate("p256");try(Native.Lease c=other.acquire()){check(Arrays.equals(Crypto.ecdh(a,c),Crypto.ecdh(c,b)),"ECDH32");}finally{other.close();}
      Native.Material noQ=Crypto.importPEM(pem("PRIVATE KEY",own.privateKey.getEncoded()));try(Native.Lease q=noQ.acquire()){check(Arrays.equals(q.publicKey.getEncoded(),own.publicKey.getEncoded()),"standaloneOmittedQ");check(Crypto.verify(q,data,Crypto.sign(q,data,"es256"),"es256"),"omittedQsign");}finally{noQ.close();}
      byte[] point=new byte[65];point[0]=4;point[64]=1;byte[] mismatch=Der.privateWithPoint(own.privateKey.getEncoded(),point);reject(()->Crypto.importPEM(pem("PRIVATE KEY",mismatch)),"mismatchedQ");
     }
    }finally{imported.close();publicOnly.close();}
   }finally{material.close();}
  }
  for(String s:new String[]{"A","Zg=","Zh==","Zg==\n","Zg== ","-_=="})check(LibEncodingBase64Decode.libEncodingBase64Decode(s)[1]!=null,"base64strict");
  for(String s:new String[]{"Zg=","Zh","Zg\n","+w","/w"})check(LibEncodingBase64UrlDecode.libEncodingBase64UrlDecode(s)[1]!=null,"base64urlstrict");
  check(((Slice)LibEncodingBase64Decode.libEncodingBase64Decode("")[0]).l==0,"emptyBase64");
  if(args.length==1)crossFixture(Path.of(args[0]));
  if(args.length==2){Native.Material fixture=Crypto.importPEM(Files.readString(Path.of(args[0]),StandardCharsets.US_ASCII));try(Native.Lease l=fixture.acquire()){check(Arrays.equals(l.publicKey.getEncoded(),Files.readAllBytes(Path.of(args[1]))),"pinnedJDKGoFixture");}finally{fixture.close();}}
  System.out.println("{\"checks\":"+checks+",\"status\":\"pass\",\"jdk\":\""+Runtime.version()+"\",\"jca\":\"SunJCE/SunRsaSign/SunEC\",\"hkdf_deriveQ\":\"BC1.86\"}");
 }
 static byte[] prop(Properties p,String key){return Base64.getDecoder().decode(p.getProperty(key));}
 static void crossFixture(Path dir)throws Exception{
  Properties props=new Properties();try(var reader=Files.newBufferedReader(dir.resolve("native.properties"),StandardCharsets.US_ASCII)){props.load(reader);}
  byte[] data=prop(props,"data");
  for(String name:new String[]{"rsa","ec"}){
   boolean ec=name.equals("ec");Native.Material material=Crypto.importPEM(Files.readString(dir.resolve(name+"-private.pem"))),certificate=Crypto.importPEM(Files.readString(dir.resolve(name+"-cert.pem")));
   try(Native.Lease own=material.acquire();Native.Lease cert=certificate.acquire()){
    check(Arrays.equals(own.publicKey.getEncoded(),cert.publicKey.getEncoded()),name+" actual certificate public identity");String[] j=Crypto.jwk(own);String canonical=ec?"{\"crv\":\""+j[1]+"\",\"kty\":\""+j[0]+"\",\"x\":\""+j[4]+"\",\"y\":\""+j[5]+"\"}":"{\"e\":\""+j[3]+"\",\"kty\":\""+j[0]+"\",\"n\":\""+j[2]+"\"}";check(Arrays.equals(canonical.getBytes(StandardCharsets.US_ASCII),prop(props,name+"-jwk")),name+" canonical Go JWK");
    byte[] signature=prop(props,name+"-signature");check(Crypto.verify(cert,data,ec?Der.rawSignature(signature,32):signature,ec?"es256":"rs256"),name+" Go signature to Java");
    byte[] signed=Crypto.sign(own,data,ec?"es256":"rs256");Files.write(dir.resolve(name+"-java-signature.bin"),ec?Der.derSignature(signed):signed);
    if(!ec){check(Arrays.equals(Crypto.rsa(own,prop(props,"rsa-cipher"),false),data),"Go OAEP to Java");Files.write(dir.resolve("rsa-java-cipher.bin"),Crypto.rsa(cert,data,true));}
    else{Native.Material peer=Crypto.importPEM(Files.readString(dir.resolve("ec-peer-public.pem")));try(Native.Lease other=peer.acquire()){byte[] secret=Crypto.ecdh(own,other);check(Arrays.equals(secret,prop(props,"ecdh")),"Go ECDH x coordinate");Files.write(dir.resolve("ecdh-java.bin"),secret);}finally{peer.close();}}
   }finally{material.close();certificate.close();}
  }
  byte[] derived=Crypto.hkdf(prop(props,"secret"),prop(props,"salt"),prop(props,"info"),42);check(Arrays.equals(derived,prop(props,"hkdf")),"Go native HKDF to Java BC");Files.write(dir.resolve("hkdf-java.bin"),derived);
  AlgorithmParameters params=AlgorithmParameters.getInstance("EC");params.init(new ECGenParameterSpec("secp256r1"));ECParameterSpec curve=params.getParameterSpec(ECParameterSpec.class);
  for(java.math.BigInteger scalar:new java.math.BigInteger[]{java.math.BigInteger.ZERO,curve.getOrder()}){PrivateKey invalid=KeyFactory.getInstance("EC").generatePrivate(new ECPrivateKeySpec(scalar,curve));reject(()->Crypto.importPEM(pem("PRIVATE KEY",invalid.getEncoded())),"EC scalar bound");}
  byte[] invalidCurve=h("308184020100301006072a8648ce3d020106052b8104000a046d306b02010104201610d53f54bd50fbfa1fbe474a7d19a2a356fec331fd487b0cb3e9c1d66c50a3a1440342000441acc6c10fb0070c29eb644b730e6c056a2c002f14043f1d6dd07f78e883ce021fddef96872cd865dddc321bcd66c0a8937d09ae62c71d591fd3472c122fe2cf");reject(()->Crypto.importPEM(pem("PRIVATE KEY",invalidCurve)),"reject wrong curve");
 }

}
