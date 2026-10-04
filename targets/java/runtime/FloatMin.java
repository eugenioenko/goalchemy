package rt;

public final class FloatMin {
 private FloatMin() {}
 public static float floatMin_f32(float a, float b) { return (float)(Floats.min(a,b)); }
 public static double floatMin_f64(double a, double b) { return (double)(Floats.min(a,b)); }
}
