namespace Rt;
public static partial class R
{
 public static void libCryptoHkdfSha256(GoTask t, Slice a0, Slice a1, Slice a2, long a3) {
  Crypto.execute(t,"hkdf",new Native.Key[]{},new object[]{a0,a1,a2,a3},new object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
