package rt;

public final class LibCryptoHmacSha256Verify {
 private LibCryptoHmacSha256Verify() {}
 public static void libCryptoHmacSha256Verify(TaskSpawn.Task t, Slice a0, Slice a1, Slice a2) {
  Crypto.execute(t,"hmac_verify",new Native.Key[]{},new Object[]{a0,a1,a2},new Object[]{false,null},"bool");
 }
}
