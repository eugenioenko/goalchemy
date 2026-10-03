// SPDX-License-Identifier: Apache-2.0
// Package encoding provides canonical RFC 4648 base64 encodings.
package encoding

import (
	"encoding/base64"
	"errors"
	"strings"
)

// MaxBytes is the largest decoded size the encoders and decoders accept (64MiB).
const MaxBytes = 64 << 20

var invalid = errors.New("encoding: invalid input or size")

// Base64Encode returns standard RFC 4648 base64 with padding.
func Base64Encode(data []byte) (string, error) {
	if len(data) > MaxBytes {
		return "", invalid
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// Base64URLEncode returns RFC 4648 base64url without padding.
func Base64URLEncode(data []byte) (string, error) {
	if len(data) > MaxBytes {
		return "", invalid
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
func decode(s string, e *base64.Encoding) ([]byte, error) {
	if len(s) > ((MaxBytes+2)/3)*4 || strings.ContainsAny(s, "\r\n\t ") {
		return nil, invalid
	}
	b, err := e.Strict().DecodeString(s)
	if err != nil || len(b) > MaxBytes || e.EncodeToString(b) != s {
		return nil, invalid
	}
	return b, nil
}

// Base64Decode strictly decodes canonical padded standard base64.
func Base64Decode(s string) ([]byte, error) { return decode(s, base64.StdEncoding) }

// Base64URLDecode strictly decodes canonical unpadded base64url.
func Base64URLDecode(s string) ([]byte, error) { return decode(s, base64.RawURLEncoding) }
