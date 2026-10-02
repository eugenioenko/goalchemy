package rt;

public final class LibCryptoEs256Verify {
 private LibCryptoEs256Verify() {}
 public static void libCryptoEs256Verify(TaskSpawn.Task t, Native.Key a0, Slice a1, Slice a2) {
  Crypto.execute(t,"es_verify",new Native.Key[]{a0},new Object[]{a1,a2},new Object[]{false,null},"bool");
 }
}
