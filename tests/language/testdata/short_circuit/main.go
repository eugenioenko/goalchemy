package main

func t(s string) bool {
	println("t", s)
	return true
}

func f(s string) bool {
	println("f", s)
	return false
}

func main() {
	println(t("a") && f("b") && t("c"))
	println(f("d") || t("e") || f("g"))
	println(!(f("h") && t("i")) || t("j"))
	x := f("k") || (t("l") && f("m"))
	println(x)
	if t("n") && (f("o") || t("p")) {
		println("yes")
	}
	for i := 0; i < 3 && t("loop"); i++ {
	}
}
