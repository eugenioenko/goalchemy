package main

type F32 float32
type Pair struct {
	X float64
	Y [1]float32
}

func identity32(x float32) float32 { return x }
func identity64(x float64) float64 { return x }

type FloatAlias = float32

func sumFloats(values ...float32) (float32, float64) {
	var sum float32
	for _, value := range values {
		sum += value
	}
	return sum, float64(sum)
}

func floatCaptures() {
	value := float32(1.5)
	closure := func() float32 { return value }
	defer func(snapshot float32) { println(snapshot) }(value)
	defer func() { println(value) }()
	value = 2.5
	println(closure())
}

func main() {
	var z float64
	var small float32
	println(z == 0, small == 0)
	// Constant folding preserves arbitrary precision until destination rounding.
	const midpoint = 0x1.00000100000000000001p0
	var c float32 = midpoint
	println(c == 0x1.000002p0, float64(midpoint) == 0x1.000001p0)
	a := identity32(16777216)
	b := identity64(16777216)
	println((a+1)-a == 0, (b+1)-b == 1, a*3 == 50331648, a/3 > 5592405)
	a++
	a--
	println(a == 16777215)
	x := int64(4611686293305294849)
	ux := uint64(9223372586610589697)
	println(float32(x) == 0x1.000002p62, float32(-x) == -0x1.000002p62, float32(ux) == 0x1.000002p63)
	println(float32(x) != float32(float64(x)), float32(ux) != float32(float64(ux)))
	f := identity64(-129.75)
	println(int8(f) == 127, uint8(f) == 127, int64(f) == -129, uint64(f) == 18446744073709551487)
	n := z / z
	inf := 1 / z
	nz := -z
	otherNaN := z / z
	println(n != otherNaN, !(n < z), !(n >= z), inf > 1e300, -inf < -1e300, 1/nz == -inf)
	println(min(n, z) != min(n, z), max(z, n) != max(z, n), 1/min(z, nz) == -inf, 1/max(z, nz) == inf)
	println(int8(n) == 0, int32(n) == -2147483648, uint32(inf) == 0, int64(inf) == -9223372036854775808, uint64(n) == 9223372036854775808)
	// Go keys containing NaN never match, even when the identical key is reused.
	m := make(map[float64]int)
	m[n] = 1
	m[n] = 2
	_, found := m[n]
	delete(m, n)
	println(len(m) == 2, !found)
	m[z] = 3
	m[nz] = 4
	println(m[z] == 4, len(m) == 3)
	k := Pair{X: n}
	mk := make(map[Pair]int)
	mk[k] = 1
	mk[k] = 2
	_, found = mk[k]
	delete(mk, k)
	println(k != k, len(mk) == 2, !found)
	var v any = n
	mi := make(map[any]int)
	mi[v] = 1
	mi[v] = 2
	_, found = mi[v]
	delete(mi, v)
	println(v != v, !found, len(mi) == 2)
	var fz any = z
	var fnz any = nz
	mi[fz] = 3
	mi[fnz] = 4
	println(fz == fnz, mi[fz] == 4, len(mi) == 3)
	var named any = F32(1.5)
	var named2 any = F32(1.5)
	println(named == named2, named.(F32) == 1.5)
	array := [2]float64{1.5, n}
	cloned := array
	cloned[0] = 7
	println(array[0] == 1.5, array != array, cloned[0] == 7)
	slice := []float32{1.5, 2.5}
	copied := make([]float32, 2)
	copy(copied, slice)
	copied[0] = 3.5
	println(slice[0] == 1.5, copied[0] == 3.5)
	var scalar float32 = 1.5
	ptr := &scalar
	*ptr = 2.5
	println(scalar == 2.5)
	// Growth must initialize spare capacity with typed positive zeros.
	grown := []float32{1, 2}
	grown = append(grown, 3)
	grown = grown[:cap(grown)]
	println(grown[3] == 0, 1/float64(grown[3]) == inf)
	grown64 := []float64{1, 2}
	grown64 = append(grown64, []float64{3}...)
	grown64 = grown64[:cap(grown64)]
	println(grown64[3] == 0, 1/grown64[3] == inf)
	grown[3] = 9
	short := grown[:2]
	short = append(short, 7)
	short = short[:cap(short)]
	println(short[3] == 9) // Reusing backing must not zero previously visible spare values.
	records := []Pair{{X: 1}, {X: 2}}
	records = append(records, Pair{X: 3})
	records = records[:cap(records)]
	println(records[3].X == 0, records[3].Y[0] == 0)
	records[3].Y[0] = 4
	println(records[2].Y[0] == 0)

	println(identity64(1551604258883524.25))
	println(float32(1.23456789), float64(1.23456789), F32(1.25), 1, true, "float")
	println(nz, float32(nz), z, float32(z), n, float32(n), inf, -inf, float32(inf), -float32(inf))
	println(float32(1e-45), float64(5e-324), float32(3.4028234663852886e38), float64(1.7976931348623157e308))
	println(float32(1e-4), float64(1e-5), float32(1e5), float64(1e6), float32(123456), float64(1234567))
	print(float32(1.5), float64(-2.25), "!\n")

	// Explicit same-width conversions form a no-fusion rounding barrier.
	left, right := identity64(0x1.0000002p0), identity64(0x1.ffffffcp-1)
	println(float64(left*right)-1 == 0)
	left32, right32 := identity32(0x1.0008p0), identity32(0x1.fffp-1)
	println(float32(left32*right32)-1 == 0)
	max32 := identity32(3.4028234663852886e38)
	min32 := identity32(0x1p-149)
	println(max32*2 == float32(inf), min32/2 == 0, 1/(-min32/2) == -float32(inf))
	println(float32(identity64(1e300)) == float32(inf), float64(min32) == 0x1p-149)
	min64 := identity64(0x1p-1074)
	println(min64/2 == 0, 1/(-min64/2) == -inf)
	positive := identity64(255.9)
	println(int8(positive) == -1, uint8(positive) == 255, int16(positive) == 255, uint16(positive) == 255,
		int32(positive) == 255, uint32(positive) == 255, int64(positive) == 255, uint64(positive) == 255)
	println(int(n) == -9223372036854775808, uint(n) == 9223372036854775808, int16(n) == 0, uint16(n) == 0)
	println(int8(-inf) == 0, uint8(-inf) == 0, int32(-inf) == -2147483648, uint32(-inf) == 0, int64(-inf) == -9223372036854775808, uint64(-inf) == 9223372036854775808)
	println(int8(identity32(255.9)) == -1, int64(identity32(-129.75)) == -129, uint64(identity32(-1)) == 18446744073709551615)
	n32 := float32(n)
	m32 := make(map[F32]int)
	m32[F32(n32)] = 1
	m32[F32(n32)] = 2
	_, found = m32[F32(n32)]
	delete(m32, F32(n32))
	println(!found, len(m32) == 2)
	m32[F32(0)] = 3
	m32[F32(float32(nz))] = 4
	println(m32[0] == 4, len(m32) == 3)
	var dynamic32 any = n32
	var dynamic64 any = n
	println(dynamic32 != dynamic32, dynamic32 != dynamic64)
	var exact32 any = float32(1.5)
	var exact64 any = float64(1.5)
	println(exact32 != exact64, exact32 != named)
	key32 := [1]float32{n32}
	ma32 := make(map[[1]float32]int)
	ma32[key32] = 1
	ma32[key32] = 2
	_, found = ma32[key32]
	delete(ma32, key32)
	println(!found, len(ma32) == 2)
	clear(slice)
	println(slice[0] == 0, 1/float64(slice[0]) == inf)
	clear(m32)
	println(len(m32) == 0)
	// Exercise host shortest-decimal formatting through runtime-produced values
	// spanning mantissas and both exponent layout branches (no constant shortcut).
	seed := uint64(123456789)
	for j := 0; j < 40; j++ {
		seed = seed*6364136223846793005 + 1442695040888963407
		value := float64(int64(seed)) / identity64(1e12)
		println(float32(value), value)
	}

	var alias FloatAlias = 1.75
	defaulted := 1.25
	var defaultInterface any = defaulted
	_, default64 := defaultInterface.(float64)
	println(alias, defaulted, default64)
	compound := identity32(1.5)
	compound += 2
	compound -= 1
	compound *= 3
	compound /= 2
	println(compound)
	sum32, sum64 := sumFloats(1.25, 2.5, 3.75)
	spread32, spread64 := sumFloats([]float32{0.5, 1.5}...)
	println(sum32, sum64, spread32, spread64)
	floatCaptures()
}
