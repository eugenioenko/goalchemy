package main

func i8(a, b int8) {
	println(a+b, a-b, a*b, a&b, a|b, a^b, a&^b, -a, ^a)
	if b != 0 {
		println(a/b, a%b)
	}
}

func u8(a, b uint8) {
	println(a+b, a-b, a*b, a&b, a|b, a^b, a&^b, -a, ^a)
	if b != 0 {
		println(a/b, a%b)
	}
}

func i32(a, b int32) {
	println(a+b, a-b, a*b, a&b, a|b, a^b, a&^b, -a, ^a)
	if b != 0 {
		println(a/b, a%b)
	}
}

func u32(a, b uint32) {
	println(a+b, a-b, a*b, a&b, a|b, a^b, a&^b, -a, ^a)
	if b != 0 {
		println(a/b, a%b)
	}
}

func i64(a, b int64) {
	println(a+b, a-b, a*b, a&b, a|b, a^b, a&^b, -a, ^a)
	if b != 0 {
		println(a/b, a%b)
	}
}

func u64(a, b uint64) {
	println(a+b, a-b, a*b, a&b, a|b, a^b, a&^b, -a, ^a)
	if b != 0 {
		println(a/b, a%b)
	}
}

func shifts(x int64, ux uint64, n uint, sn int) {
	println(x<<n, x>>n, ux<<n, ux>>n, x<<sn, x>>sn)
	var a int8 = -100
	var b uint16 = 0xffff
	var c int32 = 1
	println(a>>n, a<<n, b>>n, b<<n, c<<n)
}

func main() {
	i8(127, 1)
	i8(-128, -1)
	i8(-7, 2)
	i8(100, 100)
	u8(255, 1)
	u8(0, 1)
	u8(200, 3)
	i32(2147483647, 1)
	i32(-2147483648, -1)
	i32(-123456789, 987654321)
	i32(65536, 65536)
	u32(4294967295, 4294967295)
	u32(3000000000, 7)
	i64(9223372036854775807, 1)
	i64(-9223372036854775808, -1)
	i64(3037000500, 3037000500)
	i64(-9, 4)
	u64(18446744073709551615, 18446744073709551615)
	u64(1<<63, 3)
	u64(12345678901234567890, 10)
	for _, n := range []uint{0, 1, 7, 8, 31, 32, 63, 64, 65, 200} {
		shifts(-123456789, 0xdeadbeefcafebabe, n, int(n))
	}
	var m int = 9223372036854775807
	var u uint = 18446744073709551615
	println(m+1, u+1, m*m, u*u, m/-1, -m-1)
	println(1 < 2, int8(-1) < int8(1), uint8(255) > uint8(1), uint64(1<<63) > uint64(1), int64(-1) < int64(0))
	println(min(3, 1, 2), max(int8(-5), 7), min(uint64(18446744073709551615), 5))
}
