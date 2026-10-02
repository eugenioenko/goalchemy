package rt

func LibCryptoPrivatePEM(t *Task, key *Key) {
	r0, r1 := key.PrivatePEM()
	t.RV = []any{r0, r1}
}
