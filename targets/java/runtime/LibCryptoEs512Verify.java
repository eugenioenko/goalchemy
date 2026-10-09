package rt;

public final class LibCryptoEs512Verify {
 private LibCryptoEs512Verify() {}
 public static void libCryptoEs512Verify(TaskSpawn.Task t, Native.Key a0, Slice a1, Slice a2) {
  Crypto.execute(t,"es512_verify",new Native.Key[]{a0},new Object[]{a1,a2},new Object[]{false,null},"bool");
 }
}
