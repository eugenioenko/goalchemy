namespace Rt;
public static partial class R
{
 public static void libCryptoGenerateP384(GoTask t) {
  Crypto.execute(t,"generate_p384",new Native.Key[]{},new object[]{},new object[]{null,null},"key");
 }
}
