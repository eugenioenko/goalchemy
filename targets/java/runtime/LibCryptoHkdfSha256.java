package rt;

public final class LibCryptoHkdfSha256 {
 private LibCryptoHkdfSha256() {}
 public static void libCryptoHkdfSha256(TaskSpawn.Task t, Slice a0, Slice a1, Slice a2, long a3) {
  Crypto.execute(t,"hkdf",new Native.Key[]{},new Object[]{a0,a1,a2,a3},new Object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
