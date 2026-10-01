package main

type Celsius int16
type Mask uint8

func main() {
	var big int64 = -1
	println(int8(big), uint8(big), int16(big), uint16(big), int32(big), uint32(big), uint64(big), uint(big))
	var u uint64 = 18446744073709551615
	println(int8(u), uint8(u), int32(u), int64(u), int(u))
	var x int = 300
	println(int8(x), uint8(x), byte(x), rune(x))
	var n int32 = -2147483648
	println(int64(n), uint32(n), int16(n), uint64(n))
	c := Celsius(-40)
	f := c*9/5 + 32
	println(c, f, int(f), Mask(f), Mask(f)+Mask(250))
	var b byte = 'A'
	b += 25
	println(b, b+10, rune(b), string(rune(b)))
	const huge = 1 << 62
	println(huge, uint64(huge)<<1, int64(huge)>>61)
}
