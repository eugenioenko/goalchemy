namespace Rt;

/// <summary>IEEE scalar rounding and the selected Go linux/amd64 integer results.</summary>
public static class Floats
{
    public static double round(double x, int bits) => bits == 32 ? (double)(float)x : x;
    public static double integerFloat(long value, bool unsigned, int bits)
    {
        bool negative = !unsigned && value < 0;
        ulong n = negative ? unchecked((ulong)-value) : unchecked((ulong)value);
        if (n == 0) return 0.0;
        int length = 0; for (ulong q=n; q != 0; q >>= 1) length++;
        int shift = Math.Max(0, length - (bits == 32 ? 24 : 53));
        ulong significand = n >> shift;
        if (shift > 0) {
            ulong remainder = n & ((1UL << shift) - 1), half = 1UL << (shift - 1);
            if (remainder > half || (remainder == half && (significand & 1) != 0)) significand++;
        }
        double result = Math.ScaleB((double)significand, shift);
        return negative ? -result : result;
    }
    public static long floatInteger(double x, int bits, bool signed)
    {
        long n;
        if (!signed && bits == 64) {
            if (!double.IsFinite(x) || x < -9223372036854775808.0 || x >= 18446744073709551616.0) return long.MinValue;
            return x >= 9223372036854775808.0 ? unchecked((long)(x - 9223372036854775808.0)) ^ long.MinValue : (long)x;
        }
        int width = bits <= 16 || (bits == 32 && signed) ? 32 : 64;
        double limit = width == 32 ? 2147483648.0 : 9223372036854775808.0;
        n = double.IsFinite(x) && x >= -limit && x < limit ? (long)x : width == 32 ? int.MinValue : long.MinValue;
        if (bits == 64) return n;
        return signed ? n << (64 - bits) >> (64 - bits) : n & ((1L << bits) - 1);
    }
    public static object key(double x) => double.IsNaN(x) ? new object() : (object)(x == 0 ? 0.0 : x);
    public static double min(double a, double b) {
        if (double.IsNaN(a) || double.IsNaN(b)) return double.NaN;
        if (a == 0 && b == 0) return BitConverter.DoubleToInt64Bits(a) < 0 || BitConverter.DoubleToInt64Bits(b) < 0 ? -0.0 : 0.0;
        return a < b ? a : b;
    }
    public static double max(double a, double b) {
        if (double.IsNaN(a) || double.IsNaN(b)) return double.NaN;
        if (a == 0 && b == 0) return BitConverter.DoubleToInt64Bits(a) >= 0 || BitConverter.DoubleToInt64Bits(b) >= 0 ? 0.0 : -0.0;
        return a > b ? a : b;
    }

    public static string print(double x, int bits) {
        x = round(x, bits);
        if (double.IsNaN(x)) return "NaN";
        if (double.IsInfinity(x)) return x < 0 ? "-Inf" : "+Inf";
        if (x == 0) return BitConverter.DoubleToInt64Bits(x) < 0 ? "-0" : "0";
        string sign = x < 0 ? "-" : "";
        x = Math.Abs(x);
        var culture = System.Globalization.CultureInfo.InvariantCulture;
        string text = "";
        for (int n=1; n <= (bits == 32 ? 9 : 17); n++) {
            text = x.ToString("E" + (n-1), culture);
            if (round(double.Parse(text, culture), bits) == x) break;
        }
        string[] parts = text.Split('E');
        string digits = parts[0].Replace(".", "").TrimEnd('0');
        int exp = int.Parse(parts[1], culture), point = exp + 1;
        if (exp < -4 || exp >= 6) return sign + digits[0] + (digits.Length > 1 ? "." + digits.Substring(1) : "") + "e" + (exp < 0 ? "-" : "+") + Math.Abs(exp).ToString("D2", culture);
        return sign + (point <= 0 ? "0." + new string('0', -point) + digits : point >= digits.Length ? digits + new string('0', point-digits.Length) : digits.Substring(0, point) + "." + digits.Substring(point));
    }
}
