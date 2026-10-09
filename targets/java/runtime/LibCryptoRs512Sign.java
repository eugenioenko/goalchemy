package rt;

public final class LibCryptoRs512Sign {
 private LibCryptoRs512Sign() {}
 public static void libCryptoRs512Sign(TaskSpawn.Task t, Native.Key a0, Slice a1) {
  Crypto.execute(t,"rs512_sign",new Native.Key[]{a0},new Object[]{a1},new Object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
