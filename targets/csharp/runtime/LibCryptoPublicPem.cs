namespace Rt;
public static partial class R
{
 public static void libCryptoPublicPem(GoTask t, Native.Key a0) {
  Crypto.execute(t,"public_pem",new Native.Key[]{a0},new object[]{},new object[]{"",null},"str");
 }
}
