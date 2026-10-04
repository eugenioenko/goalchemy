from ..types.float import round_float, float_div, float_min, float_max
def float_min_f32(a, b):
    return round_float(float_min(a,b), 32)

def float_min_f64(a, b):
    return round_float(float_min(a,b), 64)
