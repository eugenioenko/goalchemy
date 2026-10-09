namespace Rt;
public static partial class R
{
 public static void libCryptoGenerateP521(GoTask t) {
  Crypto.execute(t,"generate_p521",new Native.Key[]{},new object[]{},new object[]{null,null},"key");
 }
}
