namespace Rt;
public static partial class R
{
 public static void libCryptoRandom(GoTask t, long a0) {
  Crypto.execute(t,"random",new Native.Key[]{},new object[]{a0},new object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
