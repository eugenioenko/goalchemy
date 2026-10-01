// Package hardening generates small, type-valid programs for differential
// testing. Every operation can be removed without breaking its dependencies,
// which lets the reducer preserve a failing observation.
package hardening

import (
	"fmt"
	"strings"
)

type Operation struct {
	Kind int `json:"kind"`
	Arg  int `json:"arg"`
}

type Program struct {
	Seed       uint64      `json:"seed"`
	Operations []Operation `json:"operations"`
}

func next(s *uint64) uint64 {
	*s ^= *s << 13
	*s ^= *s >> 7
	*s ^= *s << 17
	return *s
}

// Generate is stable across Go releases: it uses a specified xorshift
// sequence rather than math/rand's implementation.
func Generate(seed uint64, count int) Program {
	state := seed
	if state == 0 {
		state = 1
	}
	p := Program{Seed: seed}
	for i := 0; i < count; i++ {
		x := next(&state)
		p.Operations = append(p.Operations, Operation{Kind: int(x % 23), Arg: int((x>>8)%97) + 1})
	}
	return p
}

// Source emits ordinary Go whose output uses only specified observations.
// In particular, it never prints slice capacity or map iteration order.
func (p Program) Source() string {
	var b strings.Builder
	b.WriteString("// goalchemy:gate cooperative\npackage main\n")
	b.WriteString("func divide(x, y int64) (z int64) { defer func() { if recover() != nil { z = 77 } }(); return x / y }\n")
	b.WriteString("func handoff(x int64) int64 { ch := make(chan int64, 1); go func() { ch <- x }(); return <-ch }\n")
	b.WriteString("func nilStore(k, v int64) (panicked bool) { defer func() { if recover() != nil { panicked = true } }(); var m map[int64]int64; m[k] = v; return false }\n")
	b.WriteString("type record struct { A int32; B uint8 }\n")
	b.WriteString("type scored interface { Score() int64 }\n")
	b.WriteString("type first struct { N int64 }; func (v first) Score() int64 { return v.N + 1 }\n")
	b.WriteString("type second struct { N int64 }; func (v second) Score() int64 { return v.N - 1 }\n")
	b.WriteString("func main() {\n")
	b.WriteString("a := int64(9223372036854775807)\n")
	b.WriteString("b := int64(-137)\n")
	b.WriteString("s := []int64{a,b}\n")
	b.WriteString("alias := s[:]\n")
	b.WriteString("m := map[int64]int64{}\n")
	b.WriteString("str := \"A\\x00\\xc3\\xa9Z\"\n")
	b.WriteString("u8 := uint8(250); i32 := int32(2147483647); u64 := uint64(18446744073709551615); u := ^uint(0)\n")
	b.WriteString("rec := record{A:i32, B:u8}; var iface scored = first{N:a}\n")
	b.WriteString("counter := int32(0); bump := func(delta int32) int32 { counter += delta; return counter }\n")
	b.WriteString("_ = a; _ = b; _ = s; _ = alias; _ = m; _ = str; _ = u8; _ = i32; _ = u64; _ = u; _ = rec; _ = iface; _ = bump\n")
	for i, op := range p.Operations {
		n := op.Arg
		switch op.Kind {
		case 0:
			fmt.Fprintf(&b, "a += int64(%d); println(%d, a)\n", n, i)
		case 1:
			fmt.Fprintf(&b, "b = b*int64(%d) - a; println(%d, b)\n", n, i)
		case 2:
			fmt.Fprintf(&b, "a ^= b << uint(%d); println(%d, a)\n", n%64, i)
		case 3:
			fmt.Fprintf(&b, "s = append(s, a+int64(%d)); println(%d, len(s), s[len(s)-1])\n", n, i)
		case 4:
			fmt.Fprintf(&b, "alias[0] = b+int64(%d); println(%d, alias[0], s[0])\n", n, i)
		case 5:
			fmt.Fprintf(&b, "m[a+int64(%d)] = b; v%d, ok%d := m[a+int64(%d)]; println(%d, v%d, ok%d, len(m))\n", n, i, i, n, i, i, i)
		case 6:
			fmt.Fprintf(&b, "delete(m, a+int64(%d)); println(%d, len(m))\n", n, i)
		case 7:
			fmt.Fprintf(&b, "println(%d, len(str), str[%d])\n", i, n%len("A\x00éZ"))
		case 8:
			fmt.Fprintf(&b, "copy(s[:2], alias); println(%d, s[0], s[1])\n", i)
		case 9:
			fmt.Fprintf(&b, "println(%d, divide(a, int64(%d)))\n", i, n%4)
		case 10:
			fmt.Fprintf(&b, "println(%d, handoff(a+int64(%d)))\n", i, n)
		case 11:
			fmt.Fprintf(&b, "println(%d, nilStore(a, b))\n", i)
		case 12:
			fmt.Fprintf(&b, "u8 += uint8(%d); println(%d, u8)\n", n, i)
		case 13:
			fmt.Fprintf(&b, "i32 = i32*int32(%d) + int32(u8); println(%d, i32)\n", n, i)
		case 14:
			fmt.Fprintf(&b, "u64 = u64*uint64(%d) + uint64(u8); println(%d, u64)\n", n, i)
		case 15:
			fmt.Fprintf(&b, "u += uint(u64) + uint(%d); println(%d, u)\n", n, i)
		case 16:
			fmt.Fprintf(&b, "i32 = int32(u64); u8 = uint8(i32); u64 = uint64(u); println(%d, i32, u8, u64)\n", i)
		case 17:
			fmt.Fprintf(&b, "u8 <<= uint(%d); u64 ^= uint64(i32) >> uint(%d); println(%d, u8, u64)\n", n%9, n%65, i)
		case 18:
			fmt.Fprintf(&b, "if i32&1 == 0 { i32 += int32(%d) } else { i32 -= int32(%d) }; println(%d, i32)\n", n, n, i)
		case 19:
			fmt.Fprintf(&b, "{ sum := int32(0); for j := int32(0); j < int32(%d); j++ { sum += i32+j }; println(%d, sum) }\n", n%5, i)
		case 20:
			fmt.Fprintf(&b, "{ copied := rec; copied.A += int32(%d); println(%d, rec == copied, rec.A, copied.A); rec = copied }\n", n, i)
		case 21:
			fmt.Fprintf(&b, "{ if %d%%2 == 0 { iface = first{N:a} } else { iface = second{N:a} }; score := iface.Score(); switch iface.(type) { case first: println(%d, score, 1); case second: println(%d, score, 2) } }\n", n, i, i)
		case 22:
			fmt.Fprintf(&b, "println(%d, bump(int32(%d)), counter)\n", i, n)
		}
	}
	b.WriteString("}\n")
	return b.String()
}

// Reduce removes chunks while mismatch still reproduces. The predicate must
// return false for infrastructure failures so they do not become regressions.
func Reduce(p Program, mismatch func(Program) bool, maxChecks int) (Program, int) {
	checks := 0
	for width := len(p.Operations) / 2; width >= 1 && checks < maxChecks; width /= 2 {
		for start := 0; start+width <= len(p.Operations) && checks < maxChecks; {
			candidate := p
			candidate.Operations = append(append([]Operation{}, p.Operations[:start]...), p.Operations[start+width:]...)
			checks++
			if mismatch(candidate) {
				p = candidate
			} else {
				start += width
			}
		}
	}
	return p, checks
}
