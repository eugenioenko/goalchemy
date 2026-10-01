package main

type Item struct {
	name string
	tags map[string]int
	next *Item
}

func build(n int) *Item {
	var head *Item
	for i := 0; i < n; i++ {
		it := &Item{name: "item" + string(rune('a'+i%26)), tags: map[string]int{"i": i}, next: head}
		head = it
	}
	return head
}

func checksum(head *Item) int {
	sum := 0
	for it := head; it != nil; it = it.next {
		sum += it.tags["i"] + len(it.name)
	}
	return sum
}

func main() {
	keep := build(2000)
	before := checksum(keep)
	garbage := 0
	for round := 0; round < 100; round++ {
		tmp := build(1000)
		s := ""
		for i := 0; i < 50; i++ {
			s += "x"
		}
		garbage += checksum(tmp)%10 + len(s)
		if checksum(keep) != before {
			println("corrupted at round", round)
			return
		}
	}
	println("keep", before, checksum(keep), "garbage", garbage)
}
