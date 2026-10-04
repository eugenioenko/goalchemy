from ..types.float import round_float, float_div, float_min, float_max
def float_max_f32(a, b):
    return round_float(float_max(a,b), 32)

def float_max_f64(a, b):
    return round_float(float_max(a,b), 64)
