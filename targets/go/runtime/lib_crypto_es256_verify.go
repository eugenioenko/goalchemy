package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoES256Verify(t *Task, key *Key, data []byte, signature []byte) {
	r0, r1 := nativecrypto.ES256Verify(key, data, signature)
	t.RV = []any{r0, r1}
}
