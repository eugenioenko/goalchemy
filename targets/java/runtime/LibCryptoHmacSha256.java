package rt;

public final class LibCryptoHmacSha256 {
 private LibCryptoHmacSha256() {}
 public static void libCryptoHmacSha256(TaskSpawn.Task t, Slice a0, Slice a1) {
  Crypto.execute(t,"hmac",new Native.Key[]{},new Object[]{a0,a1},new Object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
