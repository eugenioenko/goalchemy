package rt;

public final class LibCryptoGenerateP521 {
 private LibCryptoGenerateP521() {}
 public static void libCryptoGenerateP521(TaskSpawn.Task t) {
  Crypto.execute(t,"generate_p521",new Native.Key[]{},new Object[]{},new Object[]{null,null},"key");
 }
}
