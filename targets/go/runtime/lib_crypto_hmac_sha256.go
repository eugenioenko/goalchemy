package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoHMACSHA256(t *Task, key []byte, data []byte) {
	r0, r1 := nativecrypto.HMACSHA256(key, data)
	t.RV = []any{r0, r1}
}
