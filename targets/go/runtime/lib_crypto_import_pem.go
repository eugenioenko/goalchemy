package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoImportPEM(t *Task, data string) {
	r0, r1 := nativecrypto.ImportPEM(data)
	if sched.library && r0 != nil {
		sched.retire = append(sched.retire, r0.Close)
	}
	t.RV = []any{r0, r1}
}
