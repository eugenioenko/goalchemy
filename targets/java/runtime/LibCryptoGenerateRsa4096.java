package rt;

public final class LibCryptoGenerateRsa4096 {
 private LibCryptoGenerateRsa4096() {}
 public static void libCryptoGenerateRsa4096(TaskSpawn.Task t) {
  Crypto.execute(t,"generate_rsa4096",new Native.Key[]{},new Object[]{},new Object[]{null,null},"key");
 }
}
