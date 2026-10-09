package main

type multi struct{}

func (multi) Error() string   { return "multi" }
func (multi) Unwrap() []error { return nil }

type single struct{}

func (single) Error() string { return "single" }
func (single) Unwrap() error { return nil }

type label struct{}

func (label) String() int { return 1 }

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
	var m, s error = multi{}, single{}
	var l any = label{}
	println(catch(func() { _ = m.(interface{ Unwrap() error }) }))
	println(catch(func() { _ = s.(interface{ Unwrap() []error }) }))
	println(catch(func() { _ = s.(interface{ Unwrap() error }) }))
	println(catch(func() { _ = l.(interface{ String() string }) }))
	println(catch(func() { _ = m.(interface{ Is(error) bool }) }))
	_, ok := m.(interface{ Unwrap() error })
	_, ok2 := s.(interface{ Unwrap() []error })
	println(ok, ok2)
}
