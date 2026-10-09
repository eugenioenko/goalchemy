package rt;

public final class LibCryptoRs384Verify {
 private LibCryptoRs384Verify() {}
 public static void libCryptoRs384Verify(TaskSpawn.Task t, Native.Key a0, Slice a1, Slice a2) {
  Crypto.execute(t,"rs384_verify",new Native.Key[]{a0},new Object[]{a1,a2},new Object[]{false,null},"bool");
 }
}
