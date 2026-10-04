#include "gx.h"
gx_V gx_float_convert(gx_V x, int bits) { return gx_float(gx_round_float(gx_f(x),bits)); }
gx_V gx_integer_float_convert(gx_V x, bool uns, int bits) { return gx_float(gx_integer_float(gx_i(x),uns,bits)); }
gx_V gx_float_integer_convert(gx_V x, int bits, bool sign) { return gx_int(gx_float_integer(gx_f(x),bits,sign)); }
