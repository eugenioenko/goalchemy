package rt;
public final class FloatConvert {
 private FloatConvert() {}
 public static double round(double x, int bits) { return Floats.round(x, bits); }
 public static double integerFloat(long x, boolean unsigned, int bits) { return Floats.integerFloat(x, unsigned, bits); }
 public static long floatInteger(double x, int bits, boolean signed) { return Floats.floatInteger(x, bits, signed); }
}
