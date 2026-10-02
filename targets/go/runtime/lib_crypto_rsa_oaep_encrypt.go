package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoRSAOAEPEncrypt(t *Task, key *Key, data []byte) {
	r0, r1 := nativecrypto.RSAOAEPEncrypt(key, data)
	t.RV = []any{r0, r1}
}
