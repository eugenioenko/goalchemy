package rt;

public final class LibCryptoAes256GcmDecrypt {
 private LibCryptoAes256GcmDecrypt() {}
 public static void libCryptoAes256GcmDecrypt(TaskSpawn.Task t, Slice a0, Slice a1, Slice a2, Slice a3) {
  Crypto.execute(t,"aes_decrypt",new Native.Key[]{},new Object[]{a0,a1,a2,a3},new Object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
