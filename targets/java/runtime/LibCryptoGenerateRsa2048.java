package rt;

public final class LibCryptoGenerateRsa2048 {
 private LibCryptoGenerateRsa2048() {}
 public static void libCryptoGenerateRsa2048(TaskSpawn.Task t) {
  Crypto.execute(t,"generate_rsa",new Native.Key[]{},new Object[]{},new Object[]{null,null},"key");
 }
}
