package rt;

public final class LibCryptoRsaOaepEncrypt {
 private LibCryptoRsaOaepEncrypt() {}
 public static void libCryptoRsaOaepEncrypt(TaskSpawn.Task t, Native.Key a0, Slice a1) {
  Crypto.execute(t,"rsa_encrypt",new Native.Key[]{a0},new Object[]{a1},new Object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
