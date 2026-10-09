package rt;

public final class LibCryptoEs384Sign {
 private LibCryptoEs384Sign() {}
 public static void libCryptoEs384Sign(TaskSpawn.Task t, Native.Key a0, Slice a1) {
  Crypto.execute(t,"es384_sign",new Native.Key[]{a0},new Object[]{a1},new Object[]{Slice.BYTE_NIL,null},"bytes");
 }
}
