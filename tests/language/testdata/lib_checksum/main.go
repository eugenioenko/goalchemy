package main

import "github.com/eugenioenko/goalchemy/lib/checksum"

// Native execution calls Go's hash/crc32; every translated target must agree.
func check(data []byte) {
	before := make([]byte, len(data))
	copy(before, data)
	println(checksum.CRC32IEEE(data))
	for i, v := range data {
		if v != before[i] {
			panic("CRC mutated input")
		}
	}
}

func main() {
	check(nil)
	check([]byte{})
	check([]byte("123456789"))
	check([]byte("The quick brown fox jumps over the lazy dog"))
	check([]byte{0, 255, 128, 1})
	var state uint32 = 0x8badf00d
	for _, n := range []int{0, 1, 2, 3, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65, 255, 256, 257, 4095, 4096, 4097, 65535, 65536, 65537, 1048576} {
		backing := make([]byte, n+22)
		for i := range backing {
			state = state*1664525 + 1013904223
			backing[i] = byte(state >> 24)
		}
		// Non-zero offsets and spare capacity must not change CRC or adjacent data.
		before := make([]byte, len(backing))
		copy(before, backing)
		check(backing[11 : 11+n])
		for i, v := range backing {
			if v != before[i] {
				panic("CRC mutated backing")
			}
		}
	}
	var array [19]byte
	for i := range array {
		array[i] = byte(i * 19)
	}
	check(array[3:17])
}
