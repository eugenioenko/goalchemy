package rt;

public final class LibCryptoRs256Verify {
 private LibCryptoRs256Verify() {}
 public static void libCryptoRs256Verify(TaskSpawn.Task t, Native.Key a0, Slice a1, Slice a2) {
  Crypto.execute(t,"rs_verify",new Native.Key[]{a0},new Object[]{a1,a2},new Object[]{false,null},"bool");
 }
}
