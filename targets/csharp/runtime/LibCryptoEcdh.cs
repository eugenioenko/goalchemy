namespace Rt;
public static partial class R
{
 public static void libCryptoEcdh(GoTask t, Native.Key a0, Native.Key a1) {
  Crypto.execute(t,"ecdh",new Native.Key[]{a0,a1},new object[]{},new object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
