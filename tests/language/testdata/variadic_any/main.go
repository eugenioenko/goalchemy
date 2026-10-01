package main

type Stringer interface{ String() string }

type ID int

func (i ID) String() string { return "ID-" + string(rune('0'+int(i))) }

func describe(args ...any) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += ","
		}
		switch v := a.(type) {
		case nil:
			out += "nil"
		case int:
			out += "int"
		case string:
			out += v
		case Stringer:
			out += v.String()
		case bool:
			if v {
				out += "T"
			} else {
				out += "F"
			}
		default:
			out += "?"
		}
	}
	return out
}

func sum(base int, nums ...int) int {
	for _, n := range nums {
		base += n
	}
	return base
}

func forward(args ...any) string { return describe(args...) }

func main() {
	println(describe(), describe(1, "x", ID(3), true, nil, []int{}))
	println(forward("a", false))
	xs := []any{ID(1), 2}
	println(describe(xs...))
	println(sum(1), sum(1, 2, 3), sum(0, []int{4, 5}...))
	var fn func(...int) int = func(v ...int) int { return len(v) }
	println(fn(), fn(1, 2))
}
