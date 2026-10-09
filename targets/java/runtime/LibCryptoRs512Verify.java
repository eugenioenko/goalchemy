package rt;

public final class LibCryptoRs512Verify {
 private LibCryptoRs512Verify() {}
 public static void libCryptoRs512Verify(TaskSpawn.Task t, Native.Key a0, Slice a1, Slice a2) {
  Crypto.execute(t,"rs512_verify",new Native.Key[]{a0},new Object[]{a1,a2},new Object[]{false,null},"bool");
 }
}
