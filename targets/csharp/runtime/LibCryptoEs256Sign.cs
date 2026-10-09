namespace Rt;
public static partial class R
{
 public static void libCryptoEs256Sign(GoTask t, Native.Key a0, Slice a1) {
  Crypto.execute(t,"es256_sign",new Native.Key[]{a0},new object[]{a1},new object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
