package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoGenerateP256(t *Task) {
	r0, r1 := nativecrypto.GenerateP256()
	t.RV = []any{r0, r1}
}
