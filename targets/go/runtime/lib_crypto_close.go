package rt

import nativecrypto "github.com/eugenioenko/goalchemy/lib/crypto"

type Key = nativecrypto.Key

func LibCryptoClose(key *Key) { key.Close() }
