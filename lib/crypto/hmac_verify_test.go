package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestHMACSHA256Verify(t *testing.T) {
	// RFC 4231 test case 1, independent published value.
	key := bytes.Repeat([]byte{0x0b}, 20)
	data := []byte("Hi There")
	mac, _ := hex.DecodeString("b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7")
	if ok, e := HMACSHA256Verify(key, data, mac); e != nil || !ok {
		t.Fatalf("valid: %v %v", ok, e)
	}

	empty, _ := hex.DecodeString("b613679a0814d9ec772f95d778c35fc5ff1697c493715653c6c712144292c5ad")
	if ok, e := HMACSHA256Verify(nil, nil, empty); e != nil || !ok {
		t.Fatalf("empty nil: %v %v", ok, e)
	}
	if ok, e := HMACSHA256Verify([]byte{}, []byte{}, empty); e != nil || !ok {
		t.Fatalf("empty slices: %v %v", ok, e)
	}
	mac[0] ^= 1
	if ok, e := HMACSHA256Verify(key, data, mac); e != nil || ok {
		t.Fatalf("invalid: %v %v", ok, e)
	}
	if ok, e := HMACSHA256Verify(key, data, mac[:31]); e == nil || ok {
		t.Fatal("malformed accepted")
	}
	if ok, e := HMACSHA256Verify(nil, nil, nil); e == nil || ok {
		t.Fatal("nil MAC accepted")
	}
	if ok, e := HMACSHA256Verify(make([]byte, MaxBytes+1), nil, mac); e == nil || ok {
		t.Fatal("key bound")
	}
	if ok, e := HMACSHA256Verify(nil, make([]byte, MaxBytes+1), mac); e == nil || ok {
		t.Fatal("data bound")
	}
	if key[0] != 0x0b || string(data) != "Hi There" || mac[0] != 0xb1 {
		t.Fatal("input mutation")
	}
}
