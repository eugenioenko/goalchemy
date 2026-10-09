package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoES512Sign(t *Task, key *Key, data []byte) {
	r0, r1 := nativecrypto.ES512Sign(key, data)
	t.RV = []any{r0, r1}
}
