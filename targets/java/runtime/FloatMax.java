package rt;

public final class FloatMax {
 private FloatMax() {}
 public static float floatMax_f32(float a, float b) { return (float)(Floats.max(a,b)); }
 public static double floatMax_f64(double a, double b) { return (double)(Floats.max(a,b)); }
}
