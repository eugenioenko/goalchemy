package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoHMACSHA256Verify(t *Task, key []byte, data []byte, mac []byte) {
	r0, r1 := nativecrypto.HMACSHA256Verify(key, data, mac)
	t.RV = []any{r0, r1}
}
