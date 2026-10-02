using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;
namespace Rt;

/// .NET8 maintained native cryptography; SHA1 OAEP, P1363 signatures, raw ECDH and native HKDF are explicit.
public static class Crypto
{
    internal static Action<string> beforeWork=operation=>{};
    public static Native.Reject invalid()=>new("crypto: invalid input or key");
    static int size(long value,int limit){if(value<0||value>limit)throw invalid();return (int)value;}
    static void bounded(params byte[][] values){foreach(var value in values)if(value.Length>Native.MAX)throw invalid();}
    public static byte[] random(long n)=>RandomNumberGenerator.GetBytes(size(n,Native.MAX));
    public static byte[] sha(byte[] data){bounded(data);return SHA256.HashData(data);}
    public static byte[] hmac(byte[] key,byte[] data){bounded(key,data);return HMACSHA256.HashData(key,data);}
    public static bool hmacVerify(byte[] key,byte[] data,byte[] expected){if(expected.Length!=32)throw invalid();return CryptographicOperations.FixedTimeEquals(hmac(key,data),expected);}
    public static byte[] hkdf(byte[] secret,byte[] salt,byte[] info,long n){bounded(secret,salt,info);int length=size(n,8160);return length==0?Array.Empty<byte>():HKDF.DeriveKey(HashAlgorithmName.SHA256,secret,length,salt,info);}
    public static byte[] aes(byte[] key,byte[] nonce,byte[] data,byte[] aad,bool encrypt)
    {
        if(key.Length!=32||nonce.Length!=12||aad.Length>Native.MAX||data.Length>(encrypt?Native.MAX:Native.MAX+16)||!encrypt&&data.Length<16)throw invalid();
        using var cipher=new AesGcm(key,16);byte[] result=new byte[encrypt?data.Length+16:data.Length-16];
        if(encrypt)cipher.Encrypt(nonce,data,result.AsSpan(0,data.Length),result.AsSpan(data.Length,16),aad);
        else try{cipher.Decrypt(nonce,data.AsSpan(0,result.Length),data.AsSpan(result.Length,16),result,aad);}catch(AuthenticationTagMismatchException){throw new Native.Reject("crypto: authentication failed");}
        return result;
    }
    static void validate(RSA rsa){var p=rsa.ExportParameters(false);if(rsa.KeySize!=2048||p.Exponent==null||p.Exponent.Length>256)throw invalid();var n=new System.Numerics.BigInteger(p.Exponent,true,true);if(n<3||n.IsEven)throw invalid();}
    static void validate(ECDsa ec)
    {
        var p=ec.ExportParameters(false);if(p.Curve.Oid.Value!="1.2.840.10045.3.1.7"||p.Q.X?.Length!=32||p.Q.Y?.Length!=32)throw invalid();
        // OpenSSL validates the point through native ECDH.
        using var own=ECDiffieHellman.Create(ECCurve.NamedCurves.nistP256);using var peer=ECDiffieHellman.Create(p);try{var value=own.DeriveRawSecretAgreement(peer.PublicKey);if(value.Length!=32)throw invalid();}catch(CryptographicException){throw invalid();}
    }
    public static Native.Material generate(bool rsa)
    {
        if(rsa){using var key=RSA.Create(2048);return new(key.ExportSubjectPublicKeyInfo(),key.ExportPkcs8PrivateKey(),false);}
        using var ec=ECDsa.Create(ECCurve.NamedCurves.nistP256);return new(ec.ExportSubjectPublicKeyInfo(),ec.ExportPkcs8PrivateKey(),true);
    }
    public static Native.Material importPEM(string binary)
    {
        if(binary.Length>65536)throw invalid();string text=Native.utf8(binary).Trim();
        if(!text.StartsWith("-----BEGIN ",StringComparison.Ordinal))throw invalid();int end=text.IndexOf("-----",11,StringComparison.Ordinal);if(end<0)throw invalid();string label=text.Substring(11,end-11),begin="-----BEGIN "+label+"-----",finish="-----END "+label+"-----";
        if(!(text.StartsWith(begin+"\n",StringComparison.Ordinal)||text.StartsWith(begin+"\r\n",StringComparison.Ordinal))||!text.EndsWith(finish,StringComparison.Ordinal)||text.IndexOf("-----BEGIN ",1,StringComparison.Ordinal)>=0)throw invalid();
        string body=text.Substring(begin.Length,text.Length-begin.Length-finish.Length);foreach(char c in body)if(!(c>='A'&&c<='Z'||c>='a'&&c<='z'||c>='0'&&c<='9'||"+/=\r\n\t ".Contains(c)))throw invalid();
        byte[] der;try{der=Convert.FromBase64String(body);var reader=new System.Formats.Asn1.AsnReader(der,System.Formats.Asn1.AsnEncodingRules.DER);reader.ReadEncodedValue();reader.ThrowIfNotEmpty();}catch(FormatException){throw invalid();}catch(System.Formats.Asn1.AsnContentException){throw invalid();}
        try
        {
            if(label=="CERTIFICATE")
            {
                using var certificate=new X509Certificate2(der);using var rsa=certificate.GetRSAPublicKey();if(rsa!=null){validate(rsa);return new(rsa.ExportSubjectPublicKeyInfo(),null,false);}using var ec=certificate.GetECDsaPublicKey();if(ec==null)throw invalid();validate(ec);return new(ec.ExportSubjectPublicKeyInfo(),null,true);
            }
            if(label!="PUBLIC KEY"&&label!="PRIVATE KEY"&&label!="RSA PUBLIC KEY"&&label!="RSA PRIVATE KEY")throw invalid();
            if(label.StartsWith("RSA ",StringComparison.Ordinal))
            {using var rsa=RSA.Create();if(label=="RSA PUBLIC KEY")rsa.ImportRSAPublicKey(der,out int used);else rsa.ImportRSAPrivateKey(der,out int used);validate(rsa);return new(rsa.ExportSubjectPublicKeyInfo(),label=="RSA PRIVATE KEY"?rsa.ExportPkcs8PrivateKey():null,false);}
            bool privateKey=label=="PRIVATE KEY";
            using(var rsa=RSA.Create())
            {
                bool imported=false;try{if(privateKey)rsa.ImportPkcs8PrivateKey(der,out int used);else rsa.ImportSubjectPublicKeyInfo(der,out int used);imported=true;}catch(CryptographicException){}
                if(imported){validate(rsa);return new(rsa.ExportSubjectPublicKeyInfo(),privateKey?rsa.ExportPkcs8PrivateKey():null,false);}
            }
            using(var ec=ECDsa.Create())
            {
                if(privateKey)ec.ImportPkcs8PrivateKey(der,out int used);else ec.ImportSubjectPublicKeyInfo(der,out int used);validate(ec);
                if(privateKey)
                {
                    var p=ec.ExportParameters(true);using var derived=ECDsa.Create(new ECParameters{Curve=ECCurve.NamedCurves.nistP256,D=p.D});var q=derived.ExportParameters(false).Q;
                    if(!CryptographicOperations.FixedTimeEquals(p.Q.X,q.X)||!CryptographicOperations.FixedTimeEquals(p.Q.Y,q.Y))throw invalid();
                    var sig=ec.SignData(new byte[]{1,2,3},HashAlgorithmName.SHA256,DSASignatureFormat.IeeeP1363FixedFieldConcatenation);if(!ec.VerifyData(new byte[]{1,2,3},sig,HashAlgorithmName.SHA256,DSASignatureFormat.IeeeP1363FixedFieldConcatenation))throw invalid();
                }
                return new(ec.ExportSubjectPublicKeyInfo(),privateKey?ec.ExportPkcs8PrivateKey():null,true);
            }
        }
        catch(CryptographicException){throw invalid();}catch(ArgumentException){throw invalid();}
    }
    static RSA rsaKey(Native.Lease key,bool privateKey)
    {if(key.ec||privateKey&&key.privateDer==null)throw invalid();var rsa=RSA.Create();try{if(privateKey)rsa.ImportPkcs8PrivateKey(key.privateDer,out _);else rsa.ImportSubjectPublicKeyInfo(key.publicDer,out _);return rsa;}catch{rsa.Dispose();throw;}}
    static ECDsa ecKey(Native.Lease key,bool privateKey)
    {if(!key.ec||privateKey&&key.privateDer==null)throw invalid();var ec=ECDsa.Create();try{if(privateKey)ec.ImportPkcs8PrivateKey(key.privateDer,out _);else ec.ImportSubjectPublicKeyInfo(key.publicDer,out _);return ec;}catch{ec.Dispose();throw;}}
    static string pem(string label,byte[] der)=>PemEncoding.WriteString(label,der)+"\n";
    public static string publicPEM(Native.Lease key)=>pem("PUBLIC KEY",key.publicDer);
    public static string privatePEM(Native.Lease key){if(key.privateDer==null)throw invalid();return pem("PRIVATE KEY",key.privateDer);}
    public static string b64url(byte[] value)=>Convert.ToBase64String(value).TrimEnd('=').Replace('+','-').Replace('/','_');
    public static string[] jwk(Native.Lease key)
    {if(key.ec){using var ec=ecKey(key,false);var p=ec.ExportParameters(false);return new[]{"EC","P-256","","",b64url(p.Q.X),b64url(p.Q.Y)};}using var rsa=rsaKey(key,false);var r=rsa.ExportParameters(false);return new[]{"RSA","",b64url(r.Modulus),b64url(r.Exponent),"",""};}
    public static byte[] rsa(Native.Lease key,byte[] data,bool encrypt)
    {if(encrypt&&data.Length>214||!encrypt&&data.Length!=256)throw invalid();using var rsa=rsaKey(key,!encrypt);try{return encrypt?rsa.Encrypt(data,RSAEncryptionPadding.OaepSHA1):rsa.Decrypt(data,RSAEncryptionPadding.OaepSHA1);}catch(CryptographicException){throw new Native.Reject("crypto: decryption failed");}}
    public static byte[] sign(Native.Lease key,byte[] data,bool ec)
    {bounded(data);if(ec){using var k=ecKey(key,true);return k.SignData(data,HashAlgorithmName.SHA256,DSASignatureFormat.IeeeP1363FixedFieldConcatenation);}using var r=rsaKey(key,true);return r.SignData(data,HashAlgorithmName.SHA256,RSASignaturePadding.Pkcs1);}
    public static bool verify(Native.Lease key,byte[] data,byte[] sig,bool ec)
    {bounded(data);if(sig.Length!=(ec?64:256))throw invalid();if(ec){using var k=ecKey(key,false);return k.VerifyData(data,sig,HashAlgorithmName.SHA256,DSASignatureFormat.IeeeP1363FixedFieldConcatenation);}using var r=rsaKey(key,false);return r.VerifyData(data,sig,HashAlgorithmName.SHA256,RSASignaturePadding.Pkcs1);}
    public static byte[] ecdh(Native.Lease own,Native.Lease peer)
    {if(!own.ec||!peer.ec||own.privateDer==null)throw invalid();using var a=ECDiffieHellman.Create();a.ImportPkcs8PrivateKey(own.privateDer,out _);using var b=ECDiffieHellman.Create();b.ImportSubjectPublicKeyInfo(peer.publicDer,out _);var secret=a.DeriveRawSecretAgreement(b.PublicKey);if(secret.Length!=32)throw new HostFault("native ECDH coordinate width");return secret;}
    public static void execute(GoTask task,string operation,Native.Key[] keys,object[] arguments,object[] zero,string output)
    {
        var leases=new Native.Lease[keys.Length];bool submitted=false;
        try
        {
            for(int i=0;i<keys.Length;i++)leases[i]=Native.acquire(keys[i]);var owned=(object[])arguments.Clone();for(int i=0;i<owned.Length;i++)if(owned[i] is Slice s)owned[i]=Native.bytes(s,operation=="aes_decrypt"&&i==2?Native.MAX+16:Native.MAX);var state=Native.state();
            Native.async(task,zero,()=>
            {
                beforeWork(operation);
                object value=operation switch
                {
                    "random"=>random((long)owned[0]),"sha"=>sha((byte[])owned[0]),"hmac"=>hmac((byte[])owned[0],(byte[])owned[1]),"hmac_verify"=>hmacVerify((byte[])owned[0],(byte[])owned[1],(byte[])owned[2]),"hkdf"=>hkdf((byte[])owned[0],(byte[])owned[1],(byte[])owned[2],(long)owned[3]),
                    "aes_encrypt" or "aes_decrypt"=>aes((byte[])owned[0],(byte[])owned[1],(byte[])owned[2],(byte[])owned[3],operation=="aes_encrypt"),"rsa_encrypt" or "rsa_decrypt"=>rsa(leases[0],(byte[])owned[0],operation=="rsa_encrypt"),
                    "rs_sign" or "es_sign"=>sign(leases[0],(byte[])owned[0],operation=="es_sign"),"rs_verify" or "es_verify"=>verify(leases[0],(byte[])owned[0],(byte[])owned[1],operation=="es_verify"),"ecdh"=>ecdh(leases[0],leases[1]),"public_pem"=>publicPEM(leases[0]),"private_pem"=>privatePEM(leases[0]),"jwk"=>jwk(leases[0]),"generate_rsa" or "generate_ec"=>state.stage(generate(operation=="generate_rsa")),"import"=>state.stage(importPEM((string)owned[0])),_=>throw new HostFault("unknown native crypto operation")
                };return new object[]{value};
            },wire=>{try{object value=output switch{"bytes"=>Native.slice((byte[])wire[0]),"strings"=>Native.strings(Array.ConvertAll((object[])wire[0], v => (string)v)),"key"=>Native.transfer(state,(long)wire[0]),_=>wire[0]};return new object[]{value,null};}catch(Native.Reject e){return Native.failure(zero,e.Message);}},leases);
            submitted=true;
        }
        catch(Native.Reject e){Native.reject(task,zero,e);}
        finally{if(!submitted)foreach(var lease in leases)lease?.Dispose();}
    }
}
