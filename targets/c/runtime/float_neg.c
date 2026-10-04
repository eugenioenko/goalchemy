#include "gx.h"
gx_V gx_float_neg_f32(gx_V a) { volatile float result = (float)(-gx_f(a)); return gx_float((double)result); }
gx_V gx_float_neg_f64(gx_V a) { volatile double result = (double)(-gx_f(a)); return gx_float((double)result); }
