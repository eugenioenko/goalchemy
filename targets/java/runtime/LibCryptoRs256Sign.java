package rt;

public final class LibCryptoRs256Sign {
 private LibCryptoRs256Sign() {}
 public static void libCryptoRs256Sign(TaskSpawn.Task t, Native.Key a0, Slice a1) {
  Crypto.execute(t,"rs_sign",new Native.Key[]{a0},new Object[]{a1},new Object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
