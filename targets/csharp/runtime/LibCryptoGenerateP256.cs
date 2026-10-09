namespace Rt;
public static partial class R
{
 public static void libCryptoGenerateP256(GoTask t) {
  Crypto.execute(t,"generate_p256",new Native.Key[]{},new object[]{},new object[]{null,null},"key");
 }
}
