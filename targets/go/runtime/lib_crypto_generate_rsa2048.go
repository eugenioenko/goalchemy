package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoGenerateRSA2048(t *Task) {
	r0, r1 := nativecrypto.GenerateRSA2048()
	t.RV = []any{r0, r1}
}
