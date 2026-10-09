package rt;

public final class LibCryptoGenerateP384 {
 private LibCryptoGenerateP384() {}
 public static void libCryptoGenerateP384(TaskSpawn.Task t) {
  Crypto.execute(t,"generate_p384",new Native.Key[]{},new Object[]{},new Object[]{null,null},"key");
 }
}
