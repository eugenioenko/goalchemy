package rt;

public final class LibCryptoRsaOaepDecrypt {
 private LibCryptoRsaOaepDecrypt() {}
 public static void libCryptoRsaOaepDecrypt(TaskSpawn.Task t, Native.Key a0, Slice a1) {
  Crypto.execute(t,"rsa_decrypt",new Native.Key[]{a0},new Object[]{a1},new Object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
