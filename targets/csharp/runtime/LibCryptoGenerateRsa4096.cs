namespace Rt;
public static partial class R
{
 public static void libCryptoGenerateRsa4096(GoTask t) {
  Crypto.execute(t,"generate_rsa4096",new Native.Key[]{},new object[]{},new object[]{null,null},"key");
 }
}
