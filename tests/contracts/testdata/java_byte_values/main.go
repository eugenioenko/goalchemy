package main

import "github.com/eugenioenko/goalchemy/lib/runtime"

type Octet uint8
type Array [3]Octet
type ArrayHolder struct {
	A Array
	B byte
}

func caught(f func()) {
	defer func() { println("panic", recover() != nil) }()
	f()
}

func suspended(a Array, b []Octet) {
	runtime.Gosched()
	println(a[0], b[0], b[:cap(b)][2])
	a[0] = 1
	b[1] = 128
	runtime.Gosched()
	println(a[0], b[1])
}

func main() {
	a := Array{255, 128, 3}
	holder := ArrayHolder{A: a, B: 128}
	holderCopy := holder
	holderMap := map[ArrayHolder]int{holder: 11}
	var holderBox any = holder
	holderInterfaces := map[any]int{holderBox: 12}
	holder.A[0] = 1
	println(holderMap[holderCopy], holderMap[holder], holderInterfaces[any(holderCopy)], holderBox == any(holderCopy), holderBox.(ArrayHolder).A[0])
	m := map[Array]int{a: 7}
	c := a
	var boxed any = a
	println(m[c], a == c, boxed == any(c))
	a[0] = 0
	println(m[c], m[a], boxed.(Array)[0])
	key := [3]byte{255, 128, 3}
	other := [3]byte{255, 128, 3}
	var first any = key
	var second any = other
	mixed := map[any]int{first: 9}
	println(first == second, mixed[second])
	key[0] = 2
	println(mixed[second], first.([3]byte)[0])
	b := make([]Octet, 2, 3)
	b[0] = 255
	suspended(c, b)
	println(b[1], c[0])
	closure := func() { runtime.Gosched(); b = append(b, 3, 4); println(b[:cap(b)][5]) }
	closure()
	var index uint64 = 1
	println(b[index])
	b[index] = 255
	println(b[index])
	var huge uint64 = 1 << 63
	caught(func() { println(b[huge]) })
	caught(func() { b[huge] = 1 })
	caught(func() { println(a[huge]) })
	caught(func() { a[huge] = 1 })
	caught(func() { println(len(b[:huge])) })
	caught(func() { println(len(make([]byte, huge))) })
}
