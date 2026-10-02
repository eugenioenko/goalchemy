namespace Rt;
public static partial class R
{
 public static void libCryptoHmacSha256Verify(GoTask t, Slice a0, Slice a1, Slice a2) {
  Crypto.execute(t,"hmac_verify",new Native.Key[]{},new object[]{a0,a1,a2},new object[]{false,null},"bool");
 }
}
