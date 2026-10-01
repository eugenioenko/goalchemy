package main

type Weekday int

const (
	Sunday Weekday = iota
	Monday
	Tuesday
)

const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
)

const big = 1 << 100
const small = big >> 98
const neg = -9223372036854775808

type Flags uint8

const (
	A Flags = 1 << iota
	B
	C
)

func main() {
	println(Sunday, Monday, Tuesday)
	println(KB, MB, GB, TB)
	println(small, neg, int64(neg)+1)
	println(A|C, B&^B, ^A)
	const s = "héllo"
	println(len(s), s[1], s)
	var w Weekday = Tuesday + 4
	println(w)
}
