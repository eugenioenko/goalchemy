package main

type IntReader interface{ Read() int }
type StringReader interface{ Read() string }
type Sizer interface{ size(scale int) int }

type A struct{}

func (A) Read() string       { return "a" }
func (A) size(scale int) int { return 2 * scale }
func (A) Close() error       { return nil }

type B struct{ n int }

func (b B) Read() int           { return b.n }
func (B) size() int             { return 1 }
func (B) Close(force bool) bool { return force }

type Wrap struct{ A }

type Both struct {
	B
	label string
}

func (w Both) Read() string { return w.label }

func describe(x any) string {
	switch v := x.(type) {
	case IntReader:
		return "int:" + itoa(v.Read())
	case StringReader:
		return "string:" + v.Read()
	case interface{ Close() error }:
		return "closer"
	default:
		return "other"
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}

func catch(f func()) (msg string) {
	defer func() {
		if r := recover(); r != nil {
			msg = r.(error).Error()
		}
	}()
	f()
	return "ok"
}

func main() {
	values := []any{A{}, B{7}, Wrap{}, Both{B{3}, "both"}, &B{9}, 5}
	for _, v := range values {
		_, isInt := v.(IntReader)
		_, isString := v.(StringReader)
		_, isSizer := v.(Sizer)
		_, closer := v.(interface{ Close() error })
		_, forced := v.(interface{ Close(bool) bool })
		println(describe(v), isInt, isString, isSizer, closer, forced)
	}
	var r StringReader = A{}
	read := r.Read
	println(read())
	var ir IntReader = B{4}
	var back any = ir
	if s, ok := back.(StringReader); ok {
		println("wrong", s.Read())
	} else {
		println("B is not a StringReader")
	}
	var sz Sizer = A{}
	println(sz.size(5))
	println(catch(func() { _ = back.(StringReader) }))
	println(catch(func() { _ = any(A{}).(IntReader) }))
	println(catch(func() { _ = any(B{}).(Sizer) }))
	println(catch(func() { _ = any(Wrap{}).(interface{ Close(bool) bool }) }))
	println(catch(func() { _ = any(Both{}).(IntReader) }))
}
