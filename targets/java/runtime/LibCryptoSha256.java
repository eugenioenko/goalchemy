package rt;

public final class LibCryptoSha256 {
 private LibCryptoSha256() {}
 public static void libCryptoSha256(TaskSpawn.Task t, Slice a0) {
  Crypto.execute(t,"sha",new Native.Key[]{},new Object[]{a0},new Object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
