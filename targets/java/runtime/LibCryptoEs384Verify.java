package rt;

public final class LibCryptoEs384Verify {
 private LibCryptoEs384Verify() {}
 public static void libCryptoEs384Verify(TaskSpawn.Task t, Native.Key a0, Slice a1, Slice a2) {
  Crypto.execute(t,"es384_verify",new Native.Key[]{a0},new Object[]{a1,a2},new Object[]{false,null},"bool");
 }
}
