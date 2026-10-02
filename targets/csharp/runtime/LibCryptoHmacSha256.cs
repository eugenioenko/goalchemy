namespace Rt;
public static partial class R
{
 public static void libCryptoHmacSha256(GoTask t, Slice a0, Slice a1) {
  Crypto.execute(t,"hmac",new Native.Key[]{},new object[]{a0,a1},new object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
