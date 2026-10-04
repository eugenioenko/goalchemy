#include "gx.h"
gx_V gx_float_max_f32(gx_V a, gx_V b) { volatile float result = (float)(gx_float_max(gx_f(a),gx_f(b))); return gx_float((double)result); }
gx_V gx_float_max_f64(gx_V a, gx_V b) { volatile double result = (double)(gx_float_max(gx_f(a),gx_f(b))); return gx_float((double)result); }
