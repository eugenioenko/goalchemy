namespace Rt;
public static partial class R {
 public static double round(double x, int bits) => Floats.round(x, bits);
 public static double integerFloat(long x, bool unsigned, int bits) => Floats.integerFloat(x, unsigned, bits);
 public static long floatInteger(double x, int bits, bool signed) => Floats.floatInteger(x, bits, signed);
}
