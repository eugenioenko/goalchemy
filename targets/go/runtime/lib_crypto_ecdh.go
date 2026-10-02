package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoECDH(t *Task, private_key *Key, public_key *Key) {
	r0, r1 := nativecrypto.ECDH(private_key, public_key)
	t.RV = []any{r0, r1}
}
