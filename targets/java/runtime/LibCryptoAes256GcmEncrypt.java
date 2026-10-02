package rt;

public final class LibCryptoAes256GcmEncrypt {
 private LibCryptoAes256GcmEncrypt() {}
 public static void libCryptoAes256GcmEncrypt(TaskSpawn.Task t, Slice a0, Slice a1, Slice a2, Slice a3) {
  Crypto.execute(t,"aes_encrypt",new Native.Key[]{},new Object[]{a0,a1,a2,a3},new Object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
