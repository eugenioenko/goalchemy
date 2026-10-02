package rt;

public final class LibCryptoImportPem {
 private LibCryptoImportPem() {}
 public static void libCryptoImportPem(TaskSpawn.Task t, String a0) {
  Crypto.execute(t,"import",new Native.Key[]{},new Object[]{a0},new Object[]{null,null},"key");
 }
}
