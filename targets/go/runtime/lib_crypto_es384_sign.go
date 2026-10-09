package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoES384Sign(t *Task, key *Key, data []byte) {
	r0, r1 := nativecrypto.ES384Sign(key, data)
	t.RV = []any{r0, r1}
}
