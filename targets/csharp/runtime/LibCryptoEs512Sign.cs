namespace Rt;
public static partial class R
{
 public static void libCryptoEs512Sign(GoTask t, Native.Key a0, Slice a1) {
  Crypto.execute(t,"es512_sign",new Native.Key[]{a0},new object[]{a1},new object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
