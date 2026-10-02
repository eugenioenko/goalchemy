package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoHKDFSHA256(t *Task, secret []byte, salt []byte, info []byte, n int) {
	r0, r1 := nativecrypto.HKDFSHA256(secret, salt, info, n)
	t.RV = []any{r0, r1}
}
