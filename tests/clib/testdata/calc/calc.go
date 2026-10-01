// Package calc is a library compiled to C and called from a C host.
package calc

import "github.com/eugenioenko/goalchemy/lib/errors"

var calls int

var greetings = map[string]string{"en": "Hello", "es": "Hola"}

func init() {
	greetings["fr"] = "Bonjour"
}

// Add returns a + b.
func Add(a, b int) int {
	calls++
	return a + b
}

// Divide divides; dividing by zero panics with a runtime error.
func Divide(a, b int) int {
	calls++
	return a / b
}

// Greet greets name in a language.
func Greet(lang, name string) string {
	calls++
	g, ok := greetings[lang]
	if !ok {
		g = "Hi"
	}
	return g + ", " + name + "!"
}

// Stats returns the number of bytes and words in s.
func Stats(s string) (bytes int, words int) {
	calls++
	in := false
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			in = false
		} else if !in {
			in = true
			words++
		}
	}
	return len(s), words
}

// Fib computes Fibonacci numbers recursively.
func Fib(n int) int {
	if n < 2 {
		return n
	}
	return Fib(n-1) + Fib(n-2)
}

// MaxUint wraps around unsigned 64-bit arithmetic.
func MaxUint() uint64 {
	var x uint64
	return x - 1
}

// IsEven reports whether n is even.
func IsEven(n int) bool { return n%2 == 0 }

// Check panics with an error value when ok is false.
func Check(ok bool) int {
	calls++
	if !ok {
		panic(errors.New("check failed"))
	}
	return 1
}

// Calls counts calls into the library.
func Calls() int { return calls }
