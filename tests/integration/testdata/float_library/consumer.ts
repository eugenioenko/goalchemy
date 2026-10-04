import * as g from './main.ts';

function check(value: boolean, label: string): void {
  if (!value) throw new Error(label);
}

for (const value of [1.25, -0, Infinity, -Infinity, NaN]) {
  check(Object.is(await g.Scalar32(value), Math.fround(value)), 'float32 IEEE scalar');
  check(Object.is(await g.Scalar64(value), value), 'float64 IEEE scalar');
}
check(await g.Scalar32(16777217) === 16777216, 'float32 input rounding');
const both = await g.Both(1.25, -0);
check(both[0] === 1.25 && Object.is(both[1], -0), 'multiple float results');
const input: g.Value = {Small: 1.25, Wide: -0, Values: [2.5], Nested: [[4.5]], Pair: [6.5, 7.5]};
const first = await g.Echo(input);
check(first.Small === 1.25 && Object.is(first.Wide, -0) && first.Values![0] === 3.5 && first.Nested![0]![0] === 6.5 && first.Pair![0] === 6.5, 'nested conversion');
check(input.Values![0] === 2.5 && input.Nested![0]![0] === 4.5, 'input ownership');
first.Values![0] = 99; first.Nested![0]![0] = 99;
const second = await g.Echo(input);
check(second.Values![0] === 3.5 && second.Nested![0]![0] === 6.5, 'result ownership');
const suspended = await g.Suspended(input);
check(suspended.Values![0] === 2.5 && suspended.Pair![1] === 7.5, 'suspension');
const owned = await g.Fixed(); owned.Values![0] = 99;
check((await g.Fixed()).Values![0] === 1.5 && owned.Values![0] === 99, 'global result ownership');
check((await g.Echo({Pair: [0, 0]})).Values === null, 'nil slice');
check((await g.Echo({Values: [], Nested: [], Pair: [0, 0]})).Values!.length === 0, 'empty slice');
for (const bad of ['1', 1n, true, {}]) {
  let rejected = false;
  try { await g.Scalar32(bad as unknown as number); } catch (error) {
    rejected = error instanceof g.LibraryError && error.kind === 'invalid_argument';
  }
  check(rejected, 'invalid scalar input');
}
let rejected = false;
try { await g.Echo({Pair: [1]}); } catch (error) {
  rejected = error instanceof g.LibraryError && error.kind === 'invalid_argument';
}
check(rejected, 'array length validation');
rejected = false;
try { await g.Echo({Pair: [0, 0], Values: [true as unknown as number]}); } catch (error) {
  rejected = error instanceof g.LibraryError && error.kind === 'invalid_argument';
}
check(rejected, 'nested scalar validation');
console.log('PASS float library');
