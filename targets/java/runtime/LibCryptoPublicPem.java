package rt;

public final class LibCryptoPublicPem {
 private LibCryptoPublicPem() {}
 public static void libCryptoPublicPem(TaskSpawn.Task t, Native.Key a0) {
  Crypto.execute(t,"public_pem",new Native.Key[]{a0},new Object[]{},new Object[]{"",null},"str");
 }
}
