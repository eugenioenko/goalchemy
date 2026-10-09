package rt;

public final class LibCryptoGenerateP256 {
 private LibCryptoGenerateP256() {}
 public static void libCryptoGenerateP256(TaskSpawn.Task t) {
  Crypto.execute(t,"generate_p256",new Native.Key[]{},new Object[]{},new Object[]{null,null},"key");
 }
}
