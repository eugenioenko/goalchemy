package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoGenerateRSA2048(t *Task) {
	r0, r1 := nativecrypto.GenerateRSA2048()
	if sched.library && r0 != nil {
		sched.retire = append(sched.retire, r0.Close)
	}
	t.RV = []any{r0, r1}
}
