namespace Rt;
public static partial class R
{
 public static void libCryptoGenerateRsa2048(GoTask t) {
  Crypto.execute(t,"generate_rsa",new Native.Key[]{},new object[]{},new object[]{null,null},"key");
 }
}
