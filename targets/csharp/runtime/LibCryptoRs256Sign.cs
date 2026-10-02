namespace Rt;
public static partial class R
{
 public static void libCryptoRs256Sign(GoTask t, Native.Key a0, Slice a1) {
  Crypto.execute(t,"rs_sign",new Native.Key[]{a0},new object[]{a1},new object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
