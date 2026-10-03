package checksum

import (
	"bytes"
	"hash/crc32"
	"math/rand"
	"testing"
)

func TestCRC32IEEE(t *testing.T) {
	for _, v := range []struct {
		data []byte
		want uint32
	}{
		{nil, 0}, {[]byte{}, 0}, {[]byte("123456789"), 0xcbf43926},
		{[]byte("The quick brown fox jumps over the lazy dog"), 0x414fa339},
		{[]byte{0, 255, 128, 1}, 0xd358432c},
	} {
		if got := CRC32IEEE(v.data); got != v.want {
			t.Fatalf("%x: got %08x, want %08x", v.data, got, v.want)
		}
	}
}

func TestCRC32IEEEOracleAndNoMutation(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	sizes := []int{0, 1, 2, 3, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65, 255, 256, 257, 4095, 4096, 4097, 65535, 65536, 65537, 1048576}
	for i := 0; i < 128; i++ {
		sizes = append(sizes, rng.Intn(65537))
	}
	for _, n := range sizes {
		backing := make([]byte, n+22)
		rng.Read(backing)
		before := append([]byte(nil), backing...)
		data := backing[11 : 11+n]
		want := crc32.ChecksumIEEE(data)
		if got := CRC32IEEE(data); got != want {
			t.Fatalf("length %d: got %08x, want %08x", n, got, want)
		}
		if !bytes.Equal(backing, before) {
			t.Fatalf("length %d: input mutated", n)
		}
	}
}
