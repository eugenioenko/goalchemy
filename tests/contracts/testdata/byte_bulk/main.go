package main

type Octet uint8
type Item struct {
	A [2]int
	B []int
}

func show(b []byte) {
	for _, v := range b {
		print(v, " ")
	}
	println()
}

func main() {
	var nilBytes []byte
	empty := make([]byte, 0)
	println(append(nilBytes, nilBytes...) == nil, append(empty, nilBytes...) == nil)
	println(copy(nilBytes, "\xff\x00"), copy(empty, []byte{1}))
	backing := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	shifted := backing[2:4:8]
	within := append(shifted, backing[1:4]...)
	show(backing)
	show(within)
	copy(backing[1:6], backing[:5])
	show(backing)
	copy(backing[:5], backing[2:7])
	show(backing)
	println(copy(backing[3:5], "\xff\x00\x80ignored"))
	show(backing)
	grown := append(backing[2:4:4], backing[:1]...)
	grown[0] = 99
	show(backing)
	show(grown)
	show(grown[:4])
	grown = append(grown, '\x00', '\xff')
	show(grown)
	fromString := append([]byte{1}, "\xff\x00\x80"...)
	show(fromString)
	named := make([]Octet, 1, 6)
	named[0] = 255
	named = append(named[1:1], []Octet{128, 0, 255}...)
	other := make([]Octet, 2)
	println(copy(other, named), other[0], other[1])
	alias := named[:1:1]
	alias = append(alias, named...)
	println(alias[0], alias[1], alias[2], alias[3])
	items := []Item{{A: [2]int{1, 2}, B: []int{3}}, {A: [2]int{4, 5}, B: []int{6}}}
	copies := append([]Item(nil), items...)
	copies[0].A[0] = 9
	copies[0].B[0] = 8
	println(items[0].A[0], items[0].B[0], copies[0].A[0])
	copy(items[1:], items)
	items[1].A[0] = 7
	println(items[0].A[0], items[1].A[0], items[1].B[0])
	large := make([]byte, 1<<20)
	for i := range large {
		large[i] = byte(i*131 + (i>>8)*17)
	}
	combined := append([]byte{255, 128}, large...)
	combined = append(combined, large...)
	clone := make([]byte, len(combined)+3)
	println(copy(clone[1:], combined), copy(clone[:3], combined))
	var sum uint64
	for _, b := range clone {
		sum += uint64(b)
	}
	println(len(combined), clone[0], clone[1], clone[2], clone[len(clone)-1], sum)
	combined[2] = 99
	println(large[0], clone[3])
}
