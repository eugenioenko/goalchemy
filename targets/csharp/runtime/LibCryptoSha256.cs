namespace Rt;
public static partial class R
{
 public static void libCryptoSha256(GoTask t, Slice a0) {
  Crypto.execute(t,"sha",new Native.Key[]{},new object[]{a0},new object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
