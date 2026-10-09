namespace Rt;
public static partial class R
{
 public static void libCryptoEs384Verify(GoTask t, Native.Key a0, Slice a1, Slice a2) {
  Crypto.execute(t,"es384_verify",new Native.Key[]{a0},new object[]{a1,a2},new object[]{false,null},"bool");
 }
}
