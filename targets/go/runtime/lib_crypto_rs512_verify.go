package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoRS512Verify(t *Task, key *Key, data []byte, signature []byte) {
	r0, r1 := nativecrypto.RS512Verify(key, data, signature)
	t.RV = []any{r0, r1}
}
