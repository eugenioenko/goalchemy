import importlib
import math
import struct
import sys
import asyncio

sys.path.insert(0, sys.argv[1])
g = importlib.import_module('probe')


def result(operation):
    return operation.result(20)


def same(a, b, width):
    return math.isnan(a) and math.isnan(b) or struct.pack(width, a) == struct.pack(width, b)


for value in [1.25, -0.0, math.inf, -math.inf, math.nan]:
    assert same(result(g.Scalar32(value)), value, '>f'), 'float32 IEEE scalar'
    assert same(result(g.Scalar64(value)), value, '>d'), 'float64 IEEE scalar'
assert result(g.Scalar32(16777217)) == 16777216, 'float32 input rounding'
both = result(g.Both(1.25, -0.0))
assert both[0] == 1.25 and same(both[1], -0.0, '>d'), 'multiple float results'
input_value = {'Small': 1.25, 'Wide': -0.0, 'Values': [2.5], 'Nested': [[4.5]], 'Pair': [6.5, 7.5]}
first = result(g.Echo(input_value))
assert first['Small'] == 1.25 and same(first['Wide'], -0.0, '>d'), 'nested scalar'
assert first['Values'] == [3.5] and first['Nested'] == [[6.5]] and first['Pair'] == [6.5, 7.5], 'nested conversion'
assert input_value['Values'] == [2.5] and input_value['Nested'] == [[4.5]], 'input ownership'
first['Values'][0] = first['Nested'][0][0] = 99
second = result(g.Echo(input_value))
assert second['Values'] == [3.5] and second['Nested'] == [[6.5]], 'result ownership'
assert result(g.Suspended(input_value))['Values'] == [2.5], 'suspension'
owned = result(g.Fixed()); owned['Values'][0] = 99
assert result(g.Fixed())['Values'][0] == 1.5 and owned['Values'][0] == 99, 'global result ownership'
assert result(g.Echo({'Pair': [0, 0]}))['Values'] is None, 'nil slice'
assert result(g.Echo({'Values': [], 'Nested': [], 'Pair': [0, 0]}))['Values'] == [], 'empty slice'

for operation in [g.Scalar32(True), g.Scalar64('1'), g.Scalar32(object()), g.Scalar64(10**500), g.Echo({'Pair': [1]}), g.Echo({'Pair': [0, 0], 'Values': [True]})]:
    try:
        result(operation)
        raise AssertionError('invalid input accepted')
    except g.main.rt.LibraryFailure as error:
        assert error.kind == 'invalid_argument', error.kind


async def check_async():
    assert (await g.Suspended(input_value))['Pair'] == [6.5, 7.5]


asyncio.run(check_async())
print('PASS float library')
