package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoRS384Sign(t *Task, key *Key, data []byte) {
	r0, r1 := nativecrypto.RS384Sign(key, data)
	t.RV = []any{r0, r1}
}
