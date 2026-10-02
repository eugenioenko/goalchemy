namespace Rt;
public static partial class R
{
 public static void libCryptoAes256GcmEncrypt(GoTask t, Slice a0, Slice a1, Slice a2, Slice a3) {
  Crypto.execute(t,"aes_encrypt",new Native.Key[]{},new object[]{a0,a1,a2,a3},new object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
