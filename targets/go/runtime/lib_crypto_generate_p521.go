package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoGenerateP521(t *Task) {
	r0, r1 := nativecrypto.GenerateP521()
	if sched.library && r0 != nil {
		sched.retire = append(sched.retire, r0.Close)
	}
	t.RV = []any{r0, r1}
}
