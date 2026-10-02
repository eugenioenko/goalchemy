package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

func LibCryptoImportPEM(t *Task, data string) {
	r0, r1 := nativecrypto.ImportPEM(data)
	t.RV = []any{r0, r1}
}
