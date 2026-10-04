package rt;

/** IEEE scalar operations independent of host integer cast saturation. */
public final class Floats {
    private Floats() {}
    public static double round(double x, int bits) { return bits == 32 ? (double)(float)x : x; }
    public static double integerFloat(long value, boolean unsigned, int bits) {
        boolean negative = !unsigned && value < 0;
        long n = negative ? -value : value;
        if (n == 0) return 0.0;
        int shift = Math.max(0, 64 - Long.numberOfLeadingZeros(n) - (bits == 32 ? 24 : 53));
        long significand = n >>> shift;
        if (shift > 0) {
            long remainder = n & ((1L << shift) - 1), half = 1L << (shift - 1);
            if (remainder > half || (remainder == half && (significand & 1) != 0)) significand++;
        }
        double result = Math.scalb((double)significand, shift);
        return negative ? -result : result;
    }
    public static long floatInteger(double x, int bits, boolean signed) {
        long n;
        if (!signed && bits == 64) {
            if (!Double.isFinite(x) || x < -0x1p63 || x >= 0x1p64) return Long.MIN_VALUE;
            return x >= 0x1p63 ? ((long)(x - 0x1p63)) ^ Long.MIN_VALUE : (long)x;
        }
        int width = bits <= 16 || (bits == 32 && signed) ? 32 : 64;
        double limit = width == 32 ? 0x1p31 : 0x1p63;
        n = Double.isFinite(x) && x >= -limit && x < limit ? (long)x : width == 32 ? Integer.MIN_VALUE : Long.MIN_VALUE;
        if (bits == 64) return n;
        return signed ? n << (64 - bits) >> (64 - bits) : n & ((1L << bits) - 1);
    }
    public static Object key(double x) { return Double.isNaN(x) ? new Object() : Double.valueOf(x == 0 ? 0 : x); }
    public static double min(double a, double b) { return Math.min(a, b); }
    public static double max(double a, double b) { return Math.max(a, b); }

    public static String print(double x, int bits) {
        x = round(x, bits);
        if (Double.isNaN(x)) return "NaN";
        if (Double.isInfinite(x)) return x < 0 ? "-Inf" : "+Inf";
        if (x == 0) return Double.doubleToRawLongBits(x) < 0 ? "-0" : "0";
        String sign = x < 0 ? "-" : "";
        x = Math.abs(x);
        java.math.BigDecimal exact = new java.math.BigDecimal(x), candidate = exact;
        for (int n=1; n <= (bits == 32 ? 9 : 17); n++) {
            candidate = exact.round(new java.math.MathContext(n, java.math.RoundingMode.HALF_EVEN));
            if (round(Double.parseDouble(candidate.toString()), bits) == x) break;
        }
        candidate = candidate.stripTrailingZeros();
        String digits = candidate.unscaledValue().toString();
        int exp = digits.length() - candidate.scale() - 1;
        if (exp < -4 || exp >= 6) return sign + digits.charAt(0) + (digits.length() > 1 ? "." + digits.substring(1) : "") + "e" + (exp < 0 ? "-" : "+") + (Math.abs(exp) < 10 ? "0" : "") + Math.abs(exp);
        int point = exp + 1;
        return sign + (point <= 0 ? "0." + "0".repeat(-point) + digits : point >= digits.length() ? digits + "0".repeat(point - digits.length()) : digits.substring(0, point) + "." + digits.substring(point));
    }
}
