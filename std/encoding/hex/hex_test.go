package hex_test

import (
	"bytes"
	stdhex "encoding/hex"
	"testing"

	"github.com/eugenioenko/goalchemy/std/encoding/hex"
)

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestMatchesStd(t *testing.T) {
	for _, s := range []string{"", "0", "00", "0g", "g0", "abc", "ABCDEF", "abcdef01", "zz", "a\x00", "1\xff", "\xff1", "12 3", "abé"} {
		g, ge := hex.DecodeString(s)
		w, we := stdhex.DecodeString(s)
		if !bytes.Equal(g, w) || errText(ge) != errText(we) {
			t.Errorf("DecodeString(%q) = %x,%v want %x,%v", s, g, ge, w, we)
		}
		ga, ge := hex.AppendDecode([]byte("x"), []byte(s))
		wa, we := stdhex.AppendDecode([]byte("x"), []byte(s))
		if !bytes.Equal(ga, wa) || errText(ge) != errText(we) {
			t.Errorf("AppendDecode(%q)", s)
		}
	}
	for i := 0; i < 256; i++ {
		b := []byte{byte(i), byte(255 - i), 0}
		if hex.EncodeToString(b) != stdhex.EncodeToString(b) || errText(hex.InvalidByteError(i)) != errText(stdhex.InvalidByteError(i)) {
			t.Fatalf("byte %d", i)
		}
	}
}
