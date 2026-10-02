package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoRSAOAEPDecrypt(t *Task, key *Key, data []byte) {
	r0, r1 := nativecrypto.RSAOAEPDecrypt(key, data)
	t.RV = []any{r0, r1}
}
