namespace Rt;
public static partial class R
{
 public static void libCryptoPublicJwk(GoTask t, Native.Key a0) {
  Crypto.execute(t,"jwk",new Native.Key[]{a0},new object[]{},new object[]{Slice.NIL,null},"strings");
 }
}
