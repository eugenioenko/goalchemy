package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoAES256GCMDecrypt(t *Task, key []byte, nonce []byte, data []byte, aad []byte) {
	r0, r1 := nativecrypto.AES256GCMDecrypt(key, nonce, data, aad)
	t.RV = []any{r0, r1}
}
