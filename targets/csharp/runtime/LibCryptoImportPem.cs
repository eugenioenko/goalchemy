namespace Rt;
public static partial class R
{
 public static void libCryptoImportPem(GoTask t, string a0) {
  Crypto.execute(t,"import",new Native.Key[]{},new object[]{a0},new object[]{null,null},"key");
 }
}
