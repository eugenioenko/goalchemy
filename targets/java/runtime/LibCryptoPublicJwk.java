package rt;

public final class LibCryptoPublicJwk {
 private LibCryptoPublicJwk() {}
 public static void libCryptoPublicJwk(TaskSpawn.Task t, Native.Key a0) {
  Crypto.execute(t,"jwk",new Native.Key[]{a0},new Object[]{},new Object[]{Slice.NIL,null},"strings");
 }
}
