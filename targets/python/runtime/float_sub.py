from ..types.float import round_float, float_div, float_min, float_max
def float_sub_f32(a, b):
    return round_float(a - b, 32)

def float_sub_f64(a, b):
    return round_float(a - b, 64)

