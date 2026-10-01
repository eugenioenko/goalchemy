package main

import (
	"errors"
	"fmt"     // want GCS002
	"strings" // want GCS002
	"unsafe"  // want GCS002
)

//go:noinline
func f() {} // want:-1 GCS001

func main() {
	fmt.Println(strings.ToUpper("x"), unsafe.Sizeof(0))
	_ = errors.Join // want GCS008
}
