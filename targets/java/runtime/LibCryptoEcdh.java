package rt;

public final class LibCryptoEcdh {
 private LibCryptoEcdh() {}
 public static void libCryptoEcdh(TaskSpawn.Task t, Native.Key a0, Native.Key a1) {
  Crypto.execute(t,"ecdh",new Native.Key[]{a0,a1},new Object[]{},new Object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
