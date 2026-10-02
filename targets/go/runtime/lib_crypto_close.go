package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

type Key = nativecrypto.Key

func LibCryptoClose(t *Task, key *Key) { key.Close(); t.RV = nil }
