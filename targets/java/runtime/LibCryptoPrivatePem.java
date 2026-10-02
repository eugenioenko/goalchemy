package rt;

public final class LibCryptoPrivatePem {
 private LibCryptoPrivatePem() {}
 public static void libCryptoPrivatePem(TaskSpawn.Task t, Native.Key a0) {
  Crypto.execute(t,"private_pem",new Native.Key[]{a0},new Object[]{},new Object[]{"",null},"str");
 }
}
