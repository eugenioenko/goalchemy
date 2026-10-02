namespace Rt;
public static partial class R
{
 public static void libCryptoPrivatePem(GoTask t, Native.Key a0) {
  Crypto.execute(t,"private_pem",new Native.Key[]{a0},new object[]{},new object[]{"",null},"str");
 }
}
