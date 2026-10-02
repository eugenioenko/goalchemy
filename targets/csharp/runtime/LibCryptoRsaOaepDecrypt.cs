namespace Rt;
public static partial class R
{
 public static void libCryptoRsaOaepDecrypt(GoTask t, Native.Key a0, Slice a1) {
  Crypto.execute(t,"rsa_decrypt",new Native.Key[]{a0},new object[]{a1},new object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
