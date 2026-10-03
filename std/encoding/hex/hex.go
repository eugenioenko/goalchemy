// Package hex encodes and decodes hexadecimal strings.
package hex

import (
	"github.com/eugenioenko/goalchemy/lib/errors"
	"github.com/eugenioenko/goalchemy/std/unicode"
	"github.com/eugenioenko/goalchemy/std/unicode/utf8"
)

const hextable = "0123456789abcdef"

// ErrLength reports an odd-length input to Decode.
var ErrLength = errors.New("encoding/hex: odd length hex string")

// InvalidByteError reports a byte that is not a hexadecimal digit.
type InvalidByteError byte

func (e InvalidByteError) Error() string {
	r := rune(e)
	s := "encoding/hex: invalid byte: U+"
	for shift := 12; shift >= 0; shift -= 4 {
		s += string("0123456789ABCDEF"[r>>uint(shift)&15])
	}
	if unicode.IsPrint(r) {
		s += " '" + string(utf8.AppendRune(nil, r)) + "'"
	}
	return s
}

// EncodedLen returns the length of an encoding of n source bytes.
func EncodedLen(n int) int { return n * 2 }

// DecodedLen returns the length of a decoding of x source bytes.
func DecodedLen(x int) int { return x / 2 }

// Encode writes the lowercase hexadecimal encoding of src into dst and
// returns EncodedLen(len(src)).
func Encode(dst, src []byte) int {
	j := 0
	for _, v := range src {
		dst[j] = hextable[v>>4]
		dst[j+1] = hextable[v&0x0f]
		j += 2
	}
	return len(src) * 2
}

// AppendEncode appends the hexadecimal encoding of src to dst.
func AppendEncode(dst, src []byte) []byte {
	for _, v := range src {
		dst = append(dst, hextable[v>>4], hextable[v&0x0f])
	}
	return dst
}

// EncodeToString returns the lowercase hexadecimal encoding of src.
func EncodeToString(src []byte) string { return string(AppendEncode(nil, src)) }

func fromHex(c byte) (byte, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// Decode decodes src into dst and returns the number of bytes written. It
// accepts upper and lower case digits.
func Decode(dst, src []byte) (int, error) {
	i, j := 0, 1
	for ; j < len(src); j += 2 {
		a, ok := fromHex(src[j-1])
		if !ok {
			return i, InvalidByteError(src[j-1])
		}
		b, ok := fromHex(src[j])
		if !ok {
			return i, InvalidByteError(src[j])
		}
		dst[i] = a<<4 | b
		i++
	}
	if len(src)%2 == 1 {
		if _, ok := fromHex(src[j-1]); !ok {
			return i, InvalidByteError(src[j-1])
		}
		return i, ErrLength
	}
	return i, nil
}

// AppendDecode appends the decoding of src to dst. On error it returns the
// bytes decoded before the error.
func AppendDecode(dst, src []byte) ([]byte, error) {
	out := make([]byte, len(src)/2)
	n, err := Decode(out, src)
	return append(dst, out[:n]...), err
}

// DecodeString returns the bytes represented by the hexadecimal string s.
func DecodeString(s string) ([]byte, error) {
	src := []byte(s)
	n, err := Decode(src, src)
	return src[:n], err
}
