// Package binary translates between unsigned integers and byte sequences,
// and encodes varints.
package binary

import "github.com/eugenioenko/goalchemy/lib/errors"

// ByteOrder reads and writes unsigned integers in a byte order.
type ByteOrder interface {
	Uint16([]byte) uint16
	Uint32([]byte) uint32
	Uint64([]byte) uint64
	PutUint16([]byte, uint16)
	PutUint32([]byte, uint32)
	PutUint64([]byte, uint64)
	String() string
}

// AppendByteOrder appends unsigned integers in a byte order.
type AppendByteOrder interface {
	AppendUint16([]byte, uint16) []byte
	AppendUint32([]byte, uint32) []byte
	AppendUint64([]byte, uint64) []byte
	String() string
}

type littleEndian struct{}

type bigEndian struct{}

// LittleEndian is the little-endian ByteOrder and AppendByteOrder.
var LittleEndian littleEndian

// BigEndian is the big-endian ByteOrder and AppendByteOrder.
var BigEndian bigEndian

func (littleEndian) Uint16(b []byte) uint16 {
	_ = b[1]
	return uint16(b[0]) | uint16(b[1])<<8
}

func (littleEndian) PutUint16(b []byte, v uint16) {
	_ = b[1]
	b[0] = byte(v)
	b[1] = byte(v >> 8)
}

func (littleEndian) AppendUint16(b []byte, v uint16) []byte {
	return append(b, byte(v), byte(v>>8))
}

func (littleEndian) Uint32(b []byte) uint32 {
	_ = b[3]
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func (littleEndian) PutUint32(b []byte, v uint32) {
	_ = b[3]
	for i := 0; i < 4; i++ {
		b[i] = byte(v >> (8 * uint(i)))
	}
}

func (littleEndian) AppendUint32(b []byte, v uint32) []byte {
	return append(b, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

func (littleEndian) Uint64(b []byte) uint64 {
	_ = b[7]
	var v uint64
	for i := 7; i >= 0; i-- {
		v = v<<8 | uint64(b[i])
	}
	return v
}

func (littleEndian) PutUint64(b []byte, v uint64) {
	_ = b[7]
	for i := 0; i < 8; i++ {
		b[i] = byte(v >> (8 * uint(i)))
	}
}

func (littleEndian) AppendUint64(b []byte, v uint64) []byte {
	for i := 0; i < 8; i++ {
		b = append(b, byte(v>>(8*uint(i))))
	}
	return b
}

func (littleEndian) String() string { return "LittleEndian" }

func (bigEndian) Uint16(b []byte) uint16 {
	_ = b[1]
	return uint16(b[1]) | uint16(b[0])<<8
}

func (bigEndian) PutUint16(b []byte, v uint16) {
	_ = b[1]
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}

func (bigEndian) AppendUint16(b []byte, v uint16) []byte {
	return append(b, byte(v>>8), byte(v))
}

func (bigEndian) Uint32(b []byte) uint32 {
	_ = b[3]
	return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
}

func (bigEndian) PutUint32(b []byte, v uint32) {
	_ = b[3]
	for i := 0; i < 4; i++ {
		b[i] = byte(v >> (8 * uint(3-i)))
	}
}

func (bigEndian) AppendUint32(b []byte, v uint32) []byte {
	return append(b, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

func (bigEndian) Uint64(b []byte) uint64 {
	_ = b[7]
	var v uint64
	for i := 0; i < 8; i++ {
		v = v<<8 | uint64(b[i])
	}
	return v
}

func (bigEndian) PutUint64(b []byte, v uint64) {
	_ = b[7]
	for i := 0; i < 8; i++ {
		b[i] = byte(v >> (8 * uint(7-i)))
	}
}

func (bigEndian) AppendUint64(b []byte, v uint64) []byte {
	for i := 7; i >= 0; i-- {
		b = append(b, byte(v>>(8*uint(i))))
	}
	return b
}

func (bigEndian) String() string { return "BigEndian" }

// Maximum encoded lengths of varints.
const (
	MaxVarintLen16 = 3
	MaxVarintLen32 = 5
	MaxVarintLen64 = 10
)

var errOverflow = errors.New("binary: varint overflows a 64-bit integer")

// AppendUvarint appends the varint encoding of x to buf.
func AppendUvarint(buf []byte, x uint64) []byte {
	for x >= 0x80 {
		buf = append(buf, byte(x)|0x80)
		x >>= 7
	}
	return append(buf, byte(x))
}

// PutUvarint encodes x into buf, which must be large enough, and returns the
// number of bytes written.
func PutUvarint(buf []byte, x uint64) int {
	i := 0
	for x >= 0x80 {
		buf[i] = byte(x) | 0x80
		x >>= 7
		i++
	}
	buf[i] = byte(x)
	return i + 1
}

// Uvarint decodes a varint from buf and returns the value and the number of
// bytes read. n == 0 means buf is too short; n < 0 means the value overflows
// 64 bits and -n bytes were read.
func Uvarint(buf []byte) (uint64, int) {
	var x uint64
	var s uint
	for i, b := range buf {
		if i == MaxVarintLen64 {
			return 0, -(i + 1)
		}
		if b < 0x80 {
			if i == MaxVarintLen64-1 && b > 1 {
				return 0, -(i + 1)
			}
			return x | uint64(b)<<s, i + 1
		}
		x |= uint64(b&0x7f) << s
		s += 7
	}
	return 0, 0
}

// AppendVarint appends the zig-zag varint encoding of x to buf.
func AppendVarint(buf []byte, x int64) []byte {
	ux := uint64(x) << 1
	if x < 0 {
		ux = ^ux
	}
	return AppendUvarint(buf, ux)
}

// PutVarint encodes x with zig-zag varint encoding into buf.
func PutVarint(buf []byte, x int64) int {
	ux := uint64(x) << 1
	if x < 0 {
		ux = ^ux
	}
	return PutUvarint(buf, ux)
}

// Varint decodes a zig-zag varint from buf like Uvarint.
func Varint(buf []byte) (int64, int) {
	ux, n := Uvarint(buf)
	x := int64(ux >> 1)
	if ux&1 != 0 {
		x = ^x
	}
	return x, n
}
