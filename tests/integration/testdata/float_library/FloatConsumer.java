package consumer;

import io.goalchemy.generated.Generated;
import io.goalchemy.runtime.Library;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeUnit;

public final class FloatConsumer {
    static void check(boolean value, String label) {
        if (!value) throw new AssertionError(label);
    }
    static <T> T get(Library.Operation<T> operation) throws Exception {
        return operation.completion().toCompletableFuture().get(20, TimeUnit.SECONDS);
    }
    public static void main(String[] args) throws Exception {
        for (double value : new double[]{1.25, -0.0, Double.POSITIVE_INFINITY, Double.NEGATIVE_INFINITY, Double.NaN}) {
            float small = get(Generated.Scalar32((float)value, null));
            double wide = get(Generated.Scalar64(value, null));
            check(Double.isNaN(value) ? Float.isNaN(small) : Float.floatToRawIntBits(small) == Float.floatToRawIntBits((float)value), "float32 IEEE scalar");
            check(Double.isNaN(value) ? Double.isNaN(wide) : Double.doubleToRawLongBits(wide) == Double.doubleToRawLongBits(value), "float64 IEEE scalar");
        }
        var both = get(Generated.Both(1.25f, -0.0, null));
        check(both.value0() == 1.25f && Double.doubleToRawLongBits(both.value1()) < 0, "multiple float results");
        var input = new Generated.Value();
        input.Small = 1.25f; input.Wide = -0.0;
        input.Values = new float[]{2.5f}; input.Nested = new double[][]{{4.5}}; input.Pair = new float[]{6.5f, 7.5f};
        var first = get(Generated.Echo(input, null));
        check(first.Small == 1.25f && Double.doubleToRawLongBits(first.Wide) < 0 && first.Values[0] == 3.5f && first.Nested[0][0] == 6.5 && first.Pair[1] == 7.5f, "nested conversion");
        check(input.Values[0] == 2.5f && input.Nested[0][0] == 4.5, "input ownership");
        first.Values[0] = 99; first.Nested[0][0] = 99;
        var second = get(Generated.Echo(input, null));
        check(second.Values[0] == 3.5f && second.Nested[0][0] == 6.5, "result ownership");
        check(get(Generated.Suspended(input, null)).Values[0] == 2.5f, "suspension");
        var owned = get(Generated.Fixed(null)); owned.Values[0] = 99;
        check(get(Generated.Fixed(null)).Values[0] == 1.5f && owned.Values[0] == 99, "global result ownership");
        var nil = new Generated.Value(); nil.Pair = new float[2];
        check(get(Generated.Echo(nil, null)).Values == null, "nil slice");
        var empty = new Generated.Value(); empty.Values = new float[0]; empty.Nested = new double[0][]; empty.Pair = new float[2];
        check(get(Generated.Echo(empty, null)).Values.length == 0, "empty slice");
        input.Pair = new float[]{1};
        try { get(Generated.Echo(input, null)); throw new AssertionError("invalid array length accepted"); }
        catch (ExecutionException error) {
            check(error.getCause() instanceof Library.Failure && ((Library.Failure)error.getCause()).kind.equals("invalid_argument"), "array length validation");
        }
        System.out.println("PASS float library");
    }
}
