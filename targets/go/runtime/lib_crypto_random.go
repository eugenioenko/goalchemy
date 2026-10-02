package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoRandom(t *Task, n int) {
	r0, r1 := nativecrypto.Random(n)
	t.RV = []any{r0, r1}
}
