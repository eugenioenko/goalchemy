using System;
using System.Threading.Tasks;
using Rt;
using Generated = Goalchemy.Generated.GoProgram;

static class FloatConsumer
{
    static void Check(bool value, string label) { if (!value) throw new Exception(label); }
    static T Get<T>(Library.Operation<T> operation) => operation.Completion.WaitAsync(TimeSpan.FromSeconds(20)).GetAwaiter().GetResult();
    static void Main()
    {
        foreach (double value in new[] { 1.25, -0.0, double.PositiveInfinity, double.NegativeInfinity, double.NaN })
        {
            float small = Get(Generated.Scalar32((float)value, null));
            double wide = Get(Generated.Scalar64(value, null));
            Check(double.IsNaN(value) ? float.IsNaN(small) : BitConverter.SingleToInt32Bits(small) == BitConverter.SingleToInt32Bits((float)value), "float32 IEEE scalar");
            Check(double.IsNaN(value) ? double.IsNaN(wide) : BitConverter.DoubleToInt64Bits(wide) == BitConverter.DoubleToInt64Bits(value), "float64 IEEE scalar");
        }
        var both = Get(Generated.Both(1.25f, -0.0, null));
        Check(both.value0 == 1.25f && BitConverter.DoubleToInt64Bits(both.value1) < 0, "multiple float results");
        var input = new Generated.Value { Small = 1.25f, Wide = -0.0, Values = new[] { 2.5f }, Nested = new[] { new[] { 4.5 } }, Pair = new[] { 6.5f, 7.5f } };
        var first = Get(Generated.Echo(input, null));
        Check(first.Small == 1.25f && BitConverter.DoubleToInt64Bits(first.Wide) < 0 && first.Values[0] == 3.5f && first.Nested[0][0] == 6.5 && first.Pair[1] == 7.5f, "nested conversion");
        Check(input.Values[0] == 2.5f && input.Nested[0][0] == 4.5, "input ownership");
        first.Values[0] = 99; first.Nested[0][0] = 99;
        var second = Get(Generated.Echo(input, null));
        Check(second.Values[0] == 3.5f && second.Nested[0][0] == 6.5, "result ownership");
        Check(Get(Generated.Suspended(input, null)).Values[0] == 2.5f, "suspension");
        var owned = Get(Generated.Fixed(null)); owned.Values[0] = 99;
        Check(Get(Generated.Fixed(null)).Values[0] == 1.5f && owned.Values[0] == 99, "global result ownership");
        Check(Get(Generated.Echo(new Generated.Value { Pair = new float[2] }, null)).Values == null, "nil slice");
        Check(Get(Generated.Echo(new Generated.Value { Values = Array.Empty<float>(), Nested = Array.Empty<double[]>(), Pair = new float[2] }, null)).Values.Length == 0, "empty slice");
        input.Pair = new[] { 1.0f };
        try { Get(Generated.Echo(input, null)); throw new Exception("invalid array length accepted"); }
        catch (Library.Failure error) { Check(error.Kind == "invalid_argument", "array length validation"); }
        Console.WriteLine("PASS float library");
    }
}
